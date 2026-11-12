package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/nicolasacchi/gx/internal/client"
)

// prMixServer serves 5 issue-endpoint items per page: numbers 1..N where every
// 3rd item is a pull request (carries a "pull_request" key). Total = 12 items.
func prMixServer(t *testing.T, pages *[]string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !strings.HasSuffix(r.URL.Path, "/issues") {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		per, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
		*pages = append(*pages, r.URL.Query().Get("page"))
		var items []string
		for n := (page-1)*per + 1; n <= page*per && n <= 12; n++ {
			if n%3 == 0 {
				items = append(items, fmt.Sprintf(`{"number":%d,"title":"pr","state":"open","pull_request":{"url":"x"}}`, n))
			} else {
				items = append(items, fmt.Sprintf(`{"number":%d,"title":"i","state":"open","pull_request":null}`, n))
			}
		}
		_, _ = w.Write([]byte("[" + strings.Join(items, ",") + "]"))
	}))
	restore := client.OverrideEndpoints(srv.URL, srv.URL)
	t.Cleanup(func() { restore(); srv.Close() })
}

func TestListRealIssuesDropsPRsAndFillsLimit(t *testing.T) {
	var pages []string
	prMixServer(t, &pages)
	data, err := listRealIssues(tc(), map[string][]string{"state": {"open"}}, 5)
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct{ Number int }
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("want 5 real issues, got %d", len(rows))
	}
	for _, r := range rows {
		if r.Number%3 == 0 {
			t.Errorf("pull request #%d leaked into the result", r.Number)
		}
	}
	if len(pages) < 2 {
		t.Errorf("expected paging to fill the limit past PRs, pages=%v", pages)
	}
}

func TestFetchFilteredIssuesExcludesPRsAndPages(t *testing.T) {
	var pages []string
	prMixServer(t, &pages)
	bulkLabel, bulkMilestone = []string{"x"}, ""
	defer func() { bulkLabel = nil }()
	nums, err := fetchFilteredIssues(tc())
	if err != nil {
		t.Fatal(err)
	}
	// 12 items, 4 are PRs (3,6,9,12) -> 8 issues, all in a single 100-page.
	if len(nums) != 8 {
		t.Fatalf("want 8 issues, got %v", nums)
	}
	for _, n := range nums {
		if n%3 == 0 {
			t.Errorf("PR #%d selected for bulk operation", n)
		}
	}
}

func TestFlattenIssuesDropsPullRequests(t *testing.T) {
	data := json.RawMessage(`[{"number":1,"title":"a","state":"open"},{"number":2,"title":"b","state":"open","pull_request":{"url":"u"}}]`)
	var rows []map[string]any
	if err := json.Unmarshal(flattenIssues(data), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["number"].(float64) != 1 {
		t.Errorf("got %v", rows)
	}
}
