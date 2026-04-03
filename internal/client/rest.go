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
	"strings"
	"time"
)

const (
	baseURL    = "https://api.github.com"
	timeout    = 30 * time.Second
	maxRetries = 3
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
	if req.Method == http.MethodPost || req.Method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}
}

func (c *Client) repoPath(path string) string {
	return fmt.Sprintf("%s/repos/%s/%s/%s", baseURL, c.owner, c.repo, strings.TrimLeft(path, "/"))
}

func (c *Client) doRequest(ctx context.Context, method, rawURL string, body any) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		// If body is already json.RawMessage, use directly
		if raw, ok := body.(json.RawMessage); ok {
			bodyReader = bytes.NewReader(raw)
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(b)
		}
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
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
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
		return nil, apiErr
	}

	if resp.StatusCode == 204 || len(respBody) == 0 {
		return json.RawMessage("null"), nil
	}

	return json.RawMessage(respBody), nil
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

// Delete performs a REST DELETE request on a repo-scoped path.
func (c *Client) Delete(ctx context.Context, path string) error {
	_, err := c.doRequest(ctx, "DELETE", c.repoPath(path), nil)
	return err
}
