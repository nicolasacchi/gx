package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/nicolasacchi/gx/internal/client"
)

// projectFieldsJSON is the canned getProjectFields response (one of each type).
const projectFieldsJSON = `{"data":{"organization":{"projectV2":{"fields":{"nodes":[
	{"id":"f_status","name":"Status","dataType":"SINGLE_SELECT","options":[{"id":"o_back","name":"Backlog"},{"id":"o_done","name":"Done"}]},
	{"id":"f_pts","name":"Story Points","dataType":"NUMBER"},
	{"id":"f_jira","name":"Jira Key","dataType":"TEXT"},
	{"id":"f_date","name":"Target date","dataType":"DATE"},
	{"id":"f_sprint","name":"Sprint","dataType":"ITERATION","configuration":{"iterations":[{"id":"i_46","title":"Sprint 46"}]}},
	{"id":"f_type","name":"Component","dataType":"SINGLE_SELECT","options":[{"id":"o_tech","name":"TECH"}]}
]}}}}}`

// newMock starts an httptest server, points the gx client at it, and registers
// cleanup. The handler routes GraphQL POSTs and REST GETs to canned responses.
func newMock(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(smartHandler(t)))
	restore := client.OverrideEndpoints(srv.URL, srv.URL)
	t.Cleanup(func() { restore(); srv.Close() })
	return srv
}

func tc() *client.Client { return client.New("tok", "1000farmacie", "1000farmacie", false) }

// smartHandler dispatches by REST path / GraphQL operation in the request body.
func smartHandler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/repos/") {
			restHandler(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		q := string(body)
		switch {
		case strings.Contains(q, "addProjectV2ItemById"):
			io.WriteString(w, `{"data":{"addProjectV2ItemById":{"item":{"id":"ITEM_1"}}}}`)
		case strings.Contains(q, "addProjectV2DraftIssue"):
			io.WriteString(w, `{"data":{"addProjectV2DraftIssue":{"projectItem":{"id":"DRAFT_1"}}}}`)
		case strings.Contains(q, "convertProjectV2DraftIssueItemToIssue"):
			io.WriteString(w, `{"data":{"convertProjectV2DraftIssueItemToIssue":{"item":{"content":{"number":99,"url":"https://x/99"}}}}}`)
		case strings.Contains(q, "updateProjectV2ItemFieldValue"):
			io.WriteString(w, `{"data":{"updateProjectV2ItemFieldValue":{"projectV2Item":{"id":"ITEM_1"}}}}`)
		case strings.Contains(q, "clearProjectV2ItemFieldValue"):
			io.WriteString(w, `{"data":{"clearProjectV2ItemFieldValue":{"projectV2Item":{"id":"ITEM_1"}}}}`)
		case strings.Contains(q, "archiveProjectV2Item"):
			io.WriteString(w, `{"data":{"archiveProjectV2Item":{"item":{"id":"ITEM_1"}}}}`)
		case strings.Contains(q, "addSubIssue"):
			io.WriteString(w, `{"data":{"addSubIssue":{"issue":{"id":"ISSUE_1"}}}}`)
		case strings.Contains(q, "transferIssue"):
			io.WriteString(w, `{"data":{"transferIssue":{"issue":{"number":5,"url":"https://x/5"}}}}`)
		case strings.Contains(q, "createLinkedBranch"):
			io.WriteString(w, `{"data":{"createLinkedBranch":{"linkedBranch":{"ref":{"name":"5-fix"}}}}}`)
		case strings.Contains(q, "pinIssue"):
			io.WriteString(w, `{"data":{"pinIssue":{"issue":{"title":"T"}}}}`)
		case strings.Contains(q, "unpinIssue"):
			io.WriteString(w, `{"data":{"unpinIssue":{"issue":{"title":"T"}}}}`)
		case strings.Contains(q, "removeSubIssue"):
			io.WriteString(w, `{"data":{"removeSubIssue":{"issue":{"id":"ISSUE_1"}}}}`)
		case strings.Contains(q, "reprioritizeSubIssue"):
			io.WriteString(w, `{"data":{"reprioritizeSubIssue":{"issue":{"id":"ISSUE_1"}}}}`)
		case strings.Contains(q, "closedByPullRequestsReferences"):
			io.WriteString(w, `{"data":{"repository":{"issue":{"closedByPullRequestsReferences":{"nodes":[]},"timelineItems":{"nodes":[]}}}}}`)
		case strings.Contains(q, "timelineItems"):
			io.WriteString(w, `{"data":{"repository":{"issue":{"timelineItems":{"nodes":[]}}}}}`)
		case strings.Contains(q, "fields(first:"):
			io.WriteString(w, projectFieldsJSON)
		case strings.Contains(q, "projectsV2(first:"):
			io.WriteString(w, `{"data":{"organization":{"projectsV2":{"nodes":[{"number":3,"title":"Tasks","url":"https://x/3","closed":false,"shortDescription":""}]}}}}`)
		case strings.Contains(q, "search("):
			io.WriteString(w, `{"data":{"search":{"issueCount":1,"nodes":[{"number":12,"title":"hit","state":"OPEN"}]}}}`)
		case strings.Contains(q, "fieldValues(first:"):
			io.WriteString(w, `{"data":{"node":{"projectItems":{"nodes":[{"project":{"number":3},"fieldValues":{"nodes":[{"field":{"name":"Status"},"name":"Backlog"},{"field":{"name":"Story Points"},"number":5}]}}]}}}}`)
		case strings.Contains(q, "projectItems(first:"):
			io.WriteString(w, `{"data":{"node":{"projectItems":{"nodes":[{"id":"ITEM_1","project":{"id":"PROJ_1"}}]}}}}`)
		case strings.Contains(q, "defaultBranchRef"):
			io.WriteString(w, `{"data":{"repository":{"defaultBranchRef":{"target":{"oid":"deadbeef"}},"issue":{"id":"ISSUE_1","title":"Fix login"}}}}`)
		case strings.Contains(q, "projectV2(number:"):
			io.WriteString(w, `{"data":{"organization":{"projectV2":{"id":"PROJ_1"}}}}`)
		case strings.Contains(q, "issue(number:"):
			io.WriteString(w, `{"data":{"repository":{"issue":{"id":"ISSUE_1"}}}}`)
		case strings.Contains(q, "repository("):
			io.WriteString(w, `{"data":{"repository":{"id":"REPO_1"}}}`)
		default:
			t.Fatalf("unrouted graphql query: %s", q)
		}
	}
}

func restHandler(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasSuffix(r.URL.Path, "/milestones"):
		if r.Method == http.MethodPost {
			io.WriteString(w, `{"number":9,"title":"new"}`)
		} else if p := r.URL.Query().Get("page"); p == "" || p == "1" {
			io.WriteString(w, `[{"number":7,"title":"v2.1"},{"number":8,"title":"Closed Epic"}]`)
		} else {
			io.WriteString(w, `[]`)
		}
	case strings.HasSuffix(r.URL.Path, "/issues"):
		if r.Method == http.MethodPost {
			io.WriteString(w, `{"number":14710,"title":"created","html_url":"https://x/14710","type":{"name":"Task"},"assignees":[]}`)
		} else {
			io.WriteString(w, `[{"number":11},{"number":12}]`)
		}
	case strings.Contains(r.URL.Path, "/issues/") && strings.HasSuffix(r.URL.Path, "/labels"):
		io.WriteString(w, `[]`)
	case strings.Contains(r.URL.Path, "/issues/") && strings.HasSuffix(r.URL.Path, "/comments"):
		if r.Method == http.MethodGet {
			io.WriteString(w, `[{"id":555,"user":{"login":"alice"},"body":"hi","created_at":"2026-05-22"}]`)
		} else {
			io.WriteString(w, `{"id":555,"body":"hi"}`)
		}
	case strings.HasSuffix(r.URL.Path, "/labels"):
		// repo-level labels (issue labels handled above)
		io.WriteString(w, `[{"name":"bug","color":"d73a4a","description":"A bug"}]`)
	case strings.Contains(r.URL.Path, "/issues/comments/"):
		io.WriteString(w, `{"id":555,"body":"edited"}`)
	case strings.Contains(r.URL.Path, "/issues/"):
		// single issue GET/PATCH
		io.WriteString(w, `{"number":123,"title":"T","state":"open","type":{"name":"Bug"},"assignees":[{"login":"alice"}]}`)
	default:
		io.WriteString(w, `{}`)
	}
}

// runGx executes the root command with args, capturing stdout. getClient needs
// --token/--owner/--repo, so callers should include them.
func runGx(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags()
	old := os.Stdout
	rp, wp, _ := os.Pipe()
	os.Stdout = wp
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	wp.Close()
	os.Stdout = old
	out, _ := io.ReadAll(rp)
	return string(out), err
}

// resetFlags zeroes the package-level flag vars that persist across cobra runs,
// so one command test doesn't leak flag values into the next.
func resetFlags() {
	issueTitle, issueBody, issueBodyFile, issueType, issueEditState = "", "", "", "", ""
	issueMilestone, issueCloseReason, issueReopenReason, issueTransferTo, issueDevelopName, issueUser = "", "", "", "", "", ""
	issueState = "open"
	issueAssignees, issueAddAssignee, issueRemAssignee, issueLabel, issueAddLabel, issueRemLabel = nil, nil, nil, nil, nil, nil
	issueParent, createProjectNum = 0, 0
	createStatus, createPriority, createIteration = "", "", ""
	createPoints = 0
	issueRemoveMilestone = false
	itemsProjectNum, itemsAddProjectNum = 0, 0
	itemsStatus, itemsPriority, itemsIteration, itemsClearField, itemsDraftTitle, itemsDraftBody = "", "", "", "", "", ""
	itemsFields, itemsValues = nil, nil
	itemsPoints = 0
	itemsAddIfMissing = false
	tokenFlag, ownerFlag, repoFlag, projectFlag, jqFlag = "", "", "", "", ""
	jsonFlag, verboseFlag, quietFlag = false, false, false
	limitFlag = 50
	commentBody, commentBodyFile = "", ""
}
