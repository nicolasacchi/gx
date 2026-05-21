package client

import (
	"fmt"
	"strings"
)

// APIError represents an error from the GitHub API.
type APIError struct {
	StatusCode int
	Message    string
	Errors     []struct {
		Message  string `json:"message"`
		Type     string `json:"type"`
		Field    string `json:"field"`
		Resource string `json:"resource"`
		Code     string `json:"code"`
	}
}

func (e *APIError) Error() string {
	if e.Message != "" {
		// 422 validation failures carry the actionable detail in Errors[], not Message
		// (which is just "Validation Failed"). Surface the first field-level reason so the
		// caller can see *what* failed — e.g. a missing required issue type.
		if e.StatusCode == 422 && len(e.Errors) > 0 {
			if detail := validationDetail(e.Errors[0]); detail != "" {
				return fmt.Sprintf("github: %d — %s (%s)", e.StatusCode, e.Message, detail)
			}
		}
		return fmt.Sprintf("github: %d — %s", e.StatusCode, e.Message)
	}
	if len(e.Errors) > 0 {
		return fmt.Sprintf("github: %d — %s", e.StatusCode, e.Errors[0].Message)
	}
	return fmt.Sprintf("github: %d", e.StatusCode)
}

// validationDetail renders the actionable part of a GitHub REST validation error element.
func validationDetail(v struct {
	Message  string `json:"message"`
	Type     string `json:"type"`
	Field    string `json:"field"`
	Resource string `json:"resource"`
	Code     string `json:"code"`
}) string {
	if v.Message != "" {
		return v.Message
	}
	var parts []string
	if v.Field != "" {
		parts = append(parts, v.Field)
	}
	if v.Code != "" {
		parts = append(parts, v.Code)
	}
	return strings.Join(parts, ": ")
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
