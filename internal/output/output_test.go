package output

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestIsJSON(t *testing.T) {
	if !IsJSON(true, "") {
		t.Error("--json should force JSON")
	}
	if !IsJSON(false, "title") {
		t.Error("--jq should force JSON")
	}
	// With no flags, depends on whether stdout is a terminal; in `go test` it's a
	// pipe, so this returns true. Just assert it doesn't panic / is deterministic here.
	_ = IsJSON(false, "")
}

func TestApplyFilter(t *testing.T) {
	data := json.RawMessage(`{"a":{"b":1},"items":[{"name":"x"},{"name":"y"}]}`)
	cases := []struct {
		path, want string
	}{
		{"a.b", "1"},
		{"items.0.name", `"x"`},
		{"nope", "null"},
		{"", `{"a":{"b":1},"items":[{"name":"x"},{"name":"y"}]}`},
	}
	for _, tc := range cases {
		got, err := ApplyFilter(data, tc.path)
		if err != nil {
			t.Errorf("ApplyFilter(%q): %v", tc.path, err)
			continue
		}
		if string(got) != tc.want {
			t.Errorf("ApplyFilter(%q)=%s want %s", tc.path, got, tc.want)
		}
	}
}

func captureStdout(fn func()) string {
	old := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp
	fn()
	wp.Close()
	os.Stdout = old
	b, _ := io.ReadAll(rp)
	return string(b)
}

func TestPrintDataJSON(t *testing.T) {
	data := json.RawMessage(`{"number":7,"title":"T"}`)
	out := captureStdout(func() { _ = PrintData("issues.get", data, true, "") })
	if !strings.Contains(out, `"number": 7`) || !strings.Contains(out, `"title": "T"`) {
		t.Errorf("expected pretty JSON, got: %s", out)
	}
}

func TestPrintDataWithFilter(t *testing.T) {
	data := json.RawMessage(`{"number":7,"title":"T"}`)
	out := captureStdout(func() { _ = PrintData("", data, true, "title") })
	if strings.TrimSpace(out) != `"T"` {
		t.Errorf("expected filtered title, got: %s", out)
	}
}

func TestPrintError(t *testing.T) {
	old := os.Stderr
	rp, wp, _ := os.Pipe()
	os.Stderr = wp
	PrintError("boom", 422)
	PrintError("plain", 0)
	wp.Close()
	os.Stderr = old
	b, _ := io.ReadAll(rp)
	out := string(b)
	if !strings.Contains(out, "error: 422 — boom") || !strings.Contains(out, "error: plain") {
		t.Errorf("unexpected error output: %s", out)
	}
}
