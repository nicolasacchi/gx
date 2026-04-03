package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// flattenIssue converts a GitHub REST API issue into a flat map.
func flattenIssue(raw json.RawMessage) map[string]any {
	var issue struct {
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		Body      string `json:"body"`
		HTMLURL   string `json:"html_url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		ClosedAt  string `json:"closed_at"`
		User      *struct{ Login string } `json:"user"`
		Assignee  *struct{ Login string } `json:"assignee"`
		Assignees []struct{ Login string } `json:"assignees"`
		Labels    []struct{ Name string } `json:"labels"`
		Milestone *struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
		} `json:"milestone"`
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
	return flat
}

// flattenIssues flattens an array of GitHub issues.
func flattenIssues(data json.RawMessage) json.RawMessage {
	var issues []json.RawMessage
	if json.Unmarshal(data, &issues) != nil {
		return data
	}

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
