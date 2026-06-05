package client

import (
	"fmt"
	"strings"
)

// ValidationError is a single element of GitHub's REST `errors` array.
type ValidationError struct {
	Message  string `json:"message"`
	Type     string `json:"type"`
	Field    string `json:"field"`
	Resource string `json:"resource"`
	Code     string `json:"code"`
}

// APIError represents an error from the GitHub API.
//
// It doubles as the transport-independent error type for client-side guards
// (e.g. the write-safety confirm gate), which set Kind/Detail/Hint and leave
// StatusCode zero. Kind lets callers and agents dispatch on the failure class
// rather than parsing the message — "write_locked" means refused, not failed.
type APIError struct {
	StatusCode int
	Message    string
	Errors     []ValidationError

	// Kind classifies non-transport errors raised before/around a request.
	// Empty for ordinary GitHub API failures. Currently: "write_locked".
	Kind   string
	Detail string // human-readable reason (used when StatusCode == 0)
	Hint   string // actionable next step, appended to Error()
}

func (e *APIError) Error() string {
	// Client-side guard errors (no HTTP status) render from Detail/Hint.
	if e.Kind != "" && e.StatusCode == 0 {
		msg := e.Detail
		if msg == "" {
			msg = e.Kind
		}
		if e.Hint != "" {
			return fmt.Sprintf("%s — %s", msg, e.Hint)
		}
		return msg
	}
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
func validationDetail(v ValidationError) string {
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
	case e.Kind == "write_locked":
		return 6 // refused for safety, not failed — matches the otx/stx write-gate contract
	case e.StatusCode == 401 || e.StatusCode == 403:
		return 3
	case e.StatusCode == 404:
		return 4
	default:
		return 1
	}
}
