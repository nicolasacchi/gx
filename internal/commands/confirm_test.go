package commands

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicolasacchi/gx/internal/client"
)

// recordingServer starts an httptest server that records "METHOD /path" for every
// request, points the gx client at it, and registers cleanup. GET .../issues
// returns a small open-issue list so fetchFilteredIssues works; all else -> {}.
func recordingServer(t *testing.T, rec *[]string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*rec = append(*rec, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/issues") {
			_, _ = w.Write([]byte(`[{"number":11},{"number":12}]`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	restore := client.OverrideEndpoints(srv.URL, srv.URL)
	t.Cleanup(func() { restore(); srv.Close() })
}

func anyPatch(rec []string) bool {
	for _, s := range rec {
		if strings.HasPrefix(s, http.MethodPatch+" ") {
			return true
		}
	}
	return false
}

func TestRequireConfirm(t *testing.T) {
	cases := []struct {
		name    string
		yes     bool
		wantErr bool
	}{
		{"refuses without --yes", false, true},
		{"proceeds with --yes", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orig := yesFlag
			t.Cleanup(func() { yesFlag = orig })
			yesFlag = tc.yes

			err := requireConfirm("closing 12 issues")
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("expected nil with --yes, got %v", err)
				}
				return
			}
			var apiErr *client.APIError
			if !errors.As(err, &apiErr) || apiErr.Kind != "write_locked" {
				t.Fatalf("expected write_locked APIError, got %v", err)
			}
			if apiErr.ExitCode() != 6 {
				t.Fatalf("write_locked must map to exit 6, got %d", apiErr.ExitCode())
			}
		})
	}
}

// TestBulkClose_NoSelectorIsRefused asserts the close-all footgun is closed:
// an unscoped `gx bulk close` (even with --yes) errors before any HTTP request.
func TestBulkClose_NoSelectorIsRefused(t *testing.T) {
	var rec []string
	recordingServer(t, &rec)

	_, err := runGx(t, "bulk", "close", "--yes", "--token", "x", "--owner", "o", "--repo", "r")
	if err == nil {
		t.Fatal("unscoped bulk close must error")
	}
	if !strings.Contains(err.Error(), "selector") {
		t.Fatalf("want a selector-required error, got %v", err)
	}
	if len(rec) != 0 {
		t.Fatalf("selector guard must refuse before any network call, got: %v", rec)
	}
}

// TestBulkClose_RefusesWithoutYes verifies a scoped bulk close still refuses
// (write_locked, exit 6) until --yes is given, and PATCHes nothing meanwhile.
func TestBulkClose_RefusesWithoutYes(t *testing.T) {
	var rec []string
	recordingServer(t, &rec)

	_, err := runGx(t, "bulk", "close", "--label", "type:bug", "--token", "x", "--owner", "o", "--repo", "r")
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.Kind != "write_locked" {
		t.Fatalf("expected write_locked refusal, got %v", err)
	}
	if anyPatch(rec) {
		t.Fatal("no issue may be PATCHed when the gate refuses")
	}
}

// TestBulkClose_DryRunSendsNoMutation: --dry-run previews and exits 0 (after the
// read that counts matches) without PATCHing anything.
func TestBulkClose_DryRunSendsNoMutation(t *testing.T) {
	var rec []string
	recordingServer(t, &rec)

	_, err := runGx(t, "bulk", "close", "--label", "type:bug", "--dry-run", "--token", "x", "--owner", "o", "--repo", "r")
	if err != nil {
		t.Fatalf("dry-run should succeed, got %v", err)
	}
	if anyPatch(rec) {
		t.Fatal("--dry-run must not PATCH anything")
	}
}

// TestBulkClose_BlankSelectorIsRefused: a degenerate selector (--label " " or
// --label ",", which cobra parses to a non-empty slice of blanks) must be
// treated as "no selector" and refused before any network call.
func TestBulkClose_BlankSelectorIsRefused(t *testing.T) {
	var rec []string
	recordingServer(t, &rec)

	for _, sel := range []string{" ", ","} {
		_, err := runGx(t, "bulk", "close", "--label", sel, "--yes", "--token", "x", "--owner", "o", "--repo", "r")
		if err == nil || !strings.Contains(err.Error(), "selector") {
			t.Fatalf("--label %q should be refused as no selector, got %v", sel, err)
		}
	}
	if len(rec) != 0 {
		t.Fatalf("blank selectors must hit no network, got: %v", rec)
	}
}
