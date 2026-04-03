package client

import "fmt"

// APIError represents an error from the GitHub API.
type APIError struct {
	StatusCode int
	Message    string
	Errors     []struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	}
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("github: %d — %s", e.StatusCode, e.Message)
	}
	if len(e.Errors) > 0 {
		return fmt.Sprintf("github: %d — %s", e.StatusCode, e.Errors[0].Message)
	}
	return fmt.Sprintf("github: %d", e.StatusCode)
}

// ExitCode returns the process exit code.
func (e *APIError) ExitCode() int {
	switch {
	case e.StatusCode == 401 || e.StatusCode == 403:
		return 3
	case e.StatusCode == 404:
		return 4
	default:
		return 1
	}
}
