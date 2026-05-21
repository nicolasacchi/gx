package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// baseURL is a var (not const) so tests can point it at an httptest server.
var baseURL = "https://api.github.com"

const (
	timeout    = 30 * time.Second
	maxRetries = 3
	maxBackoff = 60 * time.Second
)

// Client talks to both GitHub REST and GraphQL APIs.
type Client struct {
	http    *http.Client
	token   string
	owner   string
	repo    string
	verbose bool
}

// New creates a GitHub API client.
func New(token, owner, repo string, verbose bool) *Client {
	return &Client{
		http:    &http.Client{Timeout: timeout},
		token:   token,
		owner:   owner,
		repo:    repo,
		verbose: verbose,
	}
}

// Owner returns the configured org/user.
func (c *Client) Owner() string { return c.owner }

// Repo returns the configured repository name.
func (c *Client) Repo() string { return c.repo }

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if req.Method == http.MethodPost || req.Method == http.MethodPatch || req.Method == http.MethodDelete {
		req.Header.Set("Content-Type", "application/json")
	}
}

func (c *Client) repoPath(path string) string {
	return fmt.Sprintf("%s/repos/%s/%s/%s", baseURL, c.owner, c.repo, strings.TrimLeft(path, "/"))
}

func (c *Client) doRequest(ctx context.Context, method, rawURL string, body any) (json.RawMessage, error) {
	// Marshal the body once so it can be re-buffered on each retry attempt
	// (an io.Reader is consumed by the first send).
	var bodyBytes []byte
	if body != nil {
		if raw, ok := body.(json.RawMessage); ok {
			bodyBytes = raw
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal body: %w", err)
			}
			bodyBytes = b
		}
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
		if err != nil {
			return nil, err
		}
		c.setHeaders(req)

		if c.verbose {
			fmt.Fprintf(os.Stderr, "→ %s %s\n", method, rawURL)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			// Transient network error (conn reset/refused, EOF). Safe to retry for
			// non-POST methods; a POST may have reached the server and created state.
			if attempt < maxRetries && method != http.MethodPost {
				lastErr = err
				delay := capDelay(time.Duration(1<<attempt) * time.Second)
				if c.verbose {
					fmt.Fprintf(os.Stderr, "  ↻ retrying after %s (network error, attempt %d/%d): %s\n", delay, attempt+1, maxRetries, err)
				}
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, err
		}
		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}

		if c.verbose {
			fmt.Fprintf(os.Stderr, "← %d (%d bytes)\n", resp.StatusCode, len(respBody))
		}

		if resp.StatusCode >= 400 {
			apiErr := &APIError{StatusCode: resp.StatusCode}
			json.Unmarshal(respBody, apiErr)
			if apiErr.Message == "" {
				apiErr.Message = string(respBody)
			}
			// Retry rate limits (any method, the request was rejected not processed)
			// and transient 5xx (skip POST to avoid duplicate creates).
			if attempt < maxRetries && isRetryable(resp, method) {
				delay := retryDelay(resp, attempt)
				if c.verbose {
					fmt.Fprintf(os.Stderr, "  ↻ retrying after %s (status %d, attempt %d/%d)\n", delay, resp.StatusCode, attempt+1, maxRetries)
				}
				lastErr = apiErr
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, apiErr
		}

		if resp.StatusCode == 204 || len(respBody) == 0 {
			return json.RawMessage("null"), nil
		}
		return json.RawMessage(respBody), nil
	}
	return nil, lastErr
}

// isRetryable reports whether a failed response should be retried. Rate limits
// (429, or 403 carrying a Retry-After) are always safe to retry because the
// request was rejected before processing. Transient 5xx are retried only for
// non-POST methods, since a POST may have partially succeeded (duplicate creates).
// A bare 403 without Retry-After (auth failure or a secondary limit that has no
// hint) is NOT retried — hammering a secondary limit only extends the block.
func isRetryable(resp *http.Response, method string) bool {
	switch {
	case resp.StatusCode == 429:
		return true
	case resp.StatusCode == 403:
		return resp.Header.Get("Retry-After") != ""
	case resp.StatusCode >= 500:
		return method != http.MethodPost
	default:
		return false
	}
}

// retryDelay computes how long to wait before the next attempt, honoring
// Retry-After (seconds) then X-RateLimit-Reset (epoch), falling back to
// exponential backoff (1s, 2s, 4s …). Always clamped to [1s, maxBackoff].
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if ra := strings.TrimSpace(resp.Header.Get("Retry-After")); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
			return capDelay(time.Duration(secs) * time.Second)
		}
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
			if epoch, err := strconv.ParseInt(reset, 10, 64); err == nil {
				if d := time.Until(time.Unix(epoch, 0)); d > 0 {
					return capDelay(d)
				}
			}
		}
	}
	return capDelay(time.Duration(1<<attempt) * time.Second)
}

func capDelay(d time.Duration) time.Duration {
	switch {
	case d > maxBackoff:
		return maxBackoff
	case d < time.Second:
		return time.Second
	default:
		return d
	}
}

// GetAbsolute performs a GET request on an absolute URL (not repo-scoped).
func (c *Client) GetAbsolute(ctx context.Context, absoluteURL string) (json.RawMessage, error) {
	return c.doRequest(ctx, "GET", absoluteURL, nil)
}

// Get performs a REST GET request on a repo-scoped path.
func (c *Client) Get(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	u := c.repoPath(path)
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return c.doRequest(ctx, "GET", u, nil)
}

// Post performs a REST POST request on a repo-scoped path.
func (c *Client) Post(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return c.doRequest(ctx, "POST", c.repoPath(path), body)
}

// Patch performs a REST PATCH request on a repo-scoped path.
func (c *Client) Patch(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return c.doRequest(ctx, "PATCH", c.repoPath(path), body)
}

// Put performs a REST PUT request on a repo-scoped path.
func (c *Client) Put(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return c.doRequest(ctx, "PUT", c.repoPath(path), body)
}

// Delete performs a REST DELETE request on a repo-scoped path.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.doRequest(ctx, "DELETE", c.repoPath(path), nil)
	return err
}

// DeleteBody performs a REST DELETE request with a JSON body (e.g. removing
// assignees, which GitHub models as DELETE .../assignees with an assignees array).
func (c *Client) DeleteBody(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return c.doRequest(ctx, "DELETE", c.repoPath(path), body)
}
