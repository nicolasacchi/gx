package client

import "testing"

func TestAPIErrorError(t *testing.T) {
	cases := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			"422 with field+code detail",
			&APIError{StatusCode: 422, Message: "Validation Failed", Errors: []ValidationError{{Field: "title", Code: "missing"}}},
			"github: 422 — Validation Failed (title: missing)",
		},
		{
			"422 with element message",
			&APIError{StatusCode: 422, Message: "Validation Failed", Errors: []ValidationError{{Message: "is invalid"}}},
			"github: 422 — Validation Failed (is invalid)",
		},
		{
			"plain non-422 message",
			&APIError{StatusCode: 404, Message: "Not Found"},
			"github: 404 — Not Found",
		},
		{
			"no message falls back to first error",
			&APIError{StatusCode: 500, Errors: []ValidationError{{Message: "boom"}}},
			"github: 500 — boom",
		},
		{
			"status only",
			&APIError{StatusCode: 418},
			"github: 418",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Error(); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestOverrideEndpoints(t *testing.T) {
	origRest, origGQL := baseURL, graphqlURL
	restore := OverrideEndpoints("http://rest.test", "http://gql.test")
	if baseURL != "http://rest.test" || graphqlURL != "http://gql.test" {
		t.Fatalf("endpoints not overridden: %s / %s", baseURL, graphqlURL)
	}
	restore()
	if baseURL != origRest || graphqlURL != origGQL {
		t.Errorf("restore failed: %s / %s", baseURL, graphqlURL)
	}
}

func TestExitCode(t *testing.T) {
	for status, want := range map[int]int{401: 2, 403: 2, 400: 3, 404: 4, 429: 5, 422: 1, 500: 1, 200: 1} {
		if got := (&APIError{StatusCode: status}).ExitCode(); got != want {
			t.Errorf("ExitCode(%d)=%d want %d", status, got, want)
		}
	}
}

func TestValidationDetail(t *testing.T) {
	if got := validationDetail(ValidationError{Message: "spelled out"}); got != "spelled out" {
		t.Errorf("message preferred: got %q", got)
	}
	if got := validationDetail(ValidationError{Field: "type", Code: "invalid"}); got != "type: invalid" {
		t.Errorf("field+code: got %q", got)
	}
	if got := validationDetail(ValidationError{Field: "type"}); got != "type" {
		t.Errorf("field only: got %q", got)
	}
}
