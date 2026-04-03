package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

const graphqlURL = "https://api.github.com/graphql"

// GraphQL executes a GraphQL query/mutation and returns the data field.
func (c *Client) GraphQL(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	body := map[string]any{"query": query}
	if variables != nil {
		body["variables"] = variables
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "→ GraphQL (%d chars)\n", len(query))
	}

	// Use doRequest with absolute URL (bypasses repo path prefix)
	resp, err := c.doRequest(ctx, "POST", graphqlURL, body)
	if err != nil {
		return nil, err
	}

	// GraphQL wraps everything in {data: ..., errors: [...]}
	var result struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return resp, nil
	}
	if len(result.Errors) > 0 {
		return nil, &APIError{
			StatusCode: 200,
			Message:    result.Errors[0].Message,
		}
	}
	return result.Data, nil
}

// IssueNodeID resolves an issue number to its GraphQL node ID.
func (c *Client) IssueNodeID(ctx context.Context, number int) (string, error) {
	query := fmt.Sprintf(`{
		repository(owner: %q, name: %q) {
			issue(number: %d) { id }
		}
	}`, c.owner, c.repo, number)

	data, err := c.GraphQL(ctx, query, nil)
	if err != nil {
		return "", err
	}

	var resp struct {
		Repository struct {
			Issue struct {
				ID string `json:"id"`
			} `json:"issue"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if resp.Repository.Issue.ID == "" {
		return "", fmt.Errorf("issue #%d not found", number)
	}
	return resp.Repository.Issue.ID, nil
}

// ProjectNodeID resolves a project number to its GraphQL node ID.
func (c *Client) ProjectNodeID(ctx context.Context, projectNumber int) (string, error) {
	query := fmt.Sprintf(`{
		organization(login: %q) {
			projectV2(number: %d) { id }
		}
	}`, c.owner, projectNumber)

	data, err := c.GraphQL(ctx, query, nil)
	if err != nil {
		return "", err
	}

	var resp struct {
		Organization struct {
			ProjectV2 struct {
				ID string `json:"id"`
			} `json:"projectV2"`
		} `json:"organization"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", err
	}
	if resp.Organization.ProjectV2.ID == "" {
		return "", fmt.Errorf("project #%d not found", projectNumber)
	}
	return resp.Organization.ProjectV2.ID, nil
}
