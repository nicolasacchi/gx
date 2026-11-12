package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nicolasacchi/gx/internal/client"
	"net/url"
	"os"
	"strconv"
)

// flattenIssue converts a GitHub REST API issue into a flat map.
func flattenIssue(raw json.RawMessage) map[string]any {
	var issue struct {
		Number    int                      `json:"number"`
		Title     string                   `json:"title"`
		State     string                   `json:"state"`
		Body      string                   `json:"body"`
		HTMLURL   string                   `json:"html_url"`
		CreatedAt string                   `json:"created_at"`
		UpdatedAt string                   `json:"updated_at"`
		ClosedAt  string                   `json:"closed_at"`
		User      *struct{ Login string }  `json:"user"`
		Assignee  *struct{ Login string }  `json:"assignee"`
		Assignees []struct{ Login string } `json:"assignees"`
		Labels    []struct{ Name string }  `json:"labels"`
		Milestone *struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
		} `json:"milestone"`
		Type *struct {
			Name string `json:"name"`
		} `json:"type"`
	}
	if json.Unmarshal(raw, &issue) != nil {
		return nil
	}

	flat := map[string]any{
		"number":     issue.Number,
		"title":      issue.Title,
		"state":      issue.State,
		"url":        issue.HTMLURL,
		"created_at": issue.CreatedAt,
		"updated_at": issue.UpdatedAt,
	}
	if issue.Body != "" {
		flat["body"] = issue.Body
	}
	if issue.ClosedAt != "" {
		flat["closed_at"] = issue.ClosedAt
	}
	if issue.User != nil {
		flat["user"] = issue.User.Login
	}
	if issue.Assignee != nil {
		flat["assignee"] = issue.Assignee.Login
	}
	if len(issue.Assignees) > 0 {
		names := make([]string, len(issue.Assignees))
		for i, a := range issue.Assignees {
			names[i] = a.Login
		}
		flat["assignees"] = names
	}
	if len(issue.Labels) > 0 {
		names := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			names[i] = l.Name
		}
		flat["labels"] = names
	}
	if issue.Milestone != nil {
		flat["milestone"] = issue.Milestone.Title
		flat["milestone_number"] = issue.Milestone.Number
	}
	if issue.Type != nil {
		flat["type"] = issue.Type.Name
	}
	return flat
}

// flattenIssues flattens an array of GitHub issues.
func flattenIssues(data json.RawMessage) json.RawMessage {
	var issues []json.RawMessage
	if json.Unmarshal(data, &issues) != nil {
		return data
	}

	issues = dropPullRequests(issues)
	fmt.Fprintf(os.Stderr, "issues: %d results\n", len(issues))

	var flattened []map[string]any
	for _, raw := range issues {
		if flat := flattenIssue(raw); flat != nil {
			flattened = append(flattened, flat)
		}
	}
	if flattened == nil {
		flattened = []map[string]any{}
	}
	out, _ := json.Marshal(flattened)
	return out
}

// parseNumber parses a string as an int (for issue/milestone numbers).
func parseNumber(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", s)
	}
	return n, nil
}

// isPullRequest reports whether a REST /issues element is really a pull request
// (the issues endpoint returns PRs too; they carry a "pull_request" key).
func isPullRequest(raw json.RawMessage) bool {
	var probe struct {
		PullRequest json.RawMessage `json:"pull_request"`
	}
	if json.Unmarshal(raw, &probe) != nil {
		return false
	}
	return len(probe.PullRequest) > 0 && string(probe.PullRequest) != "null"
}

// dropPullRequests removes pull requests from a list of REST issue objects.
func dropPullRequests(items []json.RawMessage) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(items))
	for _, it := range items {
		if !isPullRequest(it) {
			out = append(out, it)
		}
	}
	return out
}

// listRealIssues pages through the REST issues endpoint, discarding pull
// requests, until limit real issues are collected (limit <= 0 means all) or
// the results run out. params must not contain page/per_page.
func listRealIssues(c *client.Client, params url.Values, limit int) (json.RawMessage, error) {
	perPage := 100
	if limit > 0 && limit < perPage {
		perPage = limit
	}
	const maxPages = 50
	var collected []json.RawMessage
	for page := 1; page <= maxPages; page++ {
		q := url.Values{}
		for k, v := range params {
			q[k] = v
		}
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))
		data, err := c.Get(context.Background(), "issues", q)
		if err != nil {
			return nil, err
		}
		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, fmt.Errorf("unexpected issues response: %w", err)
		}
		collected = append(collected, dropPullRequests(items)...)
		if limit > 0 && len(collected) >= limit {
			collected = collected[:limit]
			break
		}
		if len(items) < perPage {
			break
		}
	}
	return json.Marshal(collected)
}
