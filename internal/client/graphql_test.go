package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func graphqlServer(t *testing.T, status int, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	old := graphqlURL
	graphqlURL = srv.URL
	t.Cleanup(func() { graphqlURL = old })
}

func TestGraphQLData(t *testing.T) {
	graphqlServer(t, 200, `{"data":{"x":1}}`)
	data, err := New("t", "o", "r", false).GraphQL(context.Background(), "{x}", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"x":1}` {
		t.Errorf("data=%s", data)
	}
}

func TestGraphQLErrors(t *testing.T) {
	graphqlServer(t, 200, `{"errors":[{"message":"Bad","path":["createIssue"]}]}`)
	_, err := New("t", "o", "r", false).GraphQL(context.Background(), "mutation{}", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*APIError)
	if !ok || ae.StatusCode != 200 {
		t.Fatalf("want *APIError{200}, got %T %v", err, err)
	}
	if ae.Message != "Bad (at createIssue)" {
		t.Errorf("message=%q", ae.Message)
	}
}

func TestGraphQLNonJSON(t *testing.T) {
	graphqlServer(t, 200, `<html>rate limited</html>`)
	if _, err := New("t", "o", "r", false).GraphQL(context.Background(), "{x}", nil); err == nil {
		t.Fatal("expected error on non-JSON 200 body")
	}
}

func TestGraphQLNullData(t *testing.T) {
	graphqlServer(t, 200, `{"data":null}`)
	if _, err := New("t", "o", "r", false).GraphQL(context.Background(), "{x}", nil); err == nil {
		t.Fatal("expected error on null data with no errors")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abc", 5); got != "abc" {
		t.Errorf("short: got %q", got)
	}
	if got := truncate("abcdef", 3); got != "abc…" {
		t.Errorf("long: got %q", got)
	}
}
