package commands

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStderr runs fn with os.Stderr redirected and returns what was written.
func captureStderr(fn func()) string {
	old := os.Stderr
	rp, wp, _ := os.Pipe()
	os.Stderr = wp
	fn()
	wp.Close()
	os.Stderr = old
	b, _ := io.ReadAll(rp)
	return string(b)
}

func TestSetFieldHelpers(t *testing.T) {
	newMock(t)
	c := tc()
	f := testFields()
	if err := setFieldValue(c, "PROJ_1", "ITEM_1", f, "Status", "Backlog"); err != nil {
		t.Errorf("setFieldValue: %v", err)
	}
	if err := setNumberField(c, "PROJ_1", "ITEM_1", f, "Story Points", 5); err != nil {
		t.Errorf("setNumberField: %v", err)
	}
	if err := setTextField(c, "PROJ_1", "ITEM_1", f, "Jira Key", "MLF-1"); err != nil {
		t.Errorf("setTextField: %v", err)
	}
	if err := setDateField(c, "PROJ_1", "ITEM_1", f, "Target date", "2026-06-01"); err != nil {
		t.Errorf("setDateField: %v", err)
	}
	if err := setIterationField(c, "PROJ_1", "ITEM_1", f, "Sprint 46"); err != nil {
		t.Errorf("setIterationField: %v", err)
	}
}

func TestGetProjectFields(t *testing.T) {
	newMock(t)
	fields, err := getProjectFields(tc(), 3)
	if err != nil {
		t.Fatalf("getProjectFields: %v", err)
	}
	if len(fields) != 6 {
		t.Fatalf("got %d fields want 6", len(fields))
	}
	if id, oid := resolveOptionID(fields, "Status", "Done"); id != "f_status" || oid != "o_done" {
		t.Errorf("Status/Done resolved to %s/%s", id, oid)
	}
	if fid, iid := resolveIterationID(fields, "Sprint 46"); fid != "f_sprint" || iid != "i_46" {
		t.Errorf("Sprint resolved to %s/%s", fid, iid)
	}
}

func TestFindProjectItemID(t *testing.T) {
	newMock(t)
	id, err := findProjectItemID(tc(), "PROJ_1", 123)
	if err != nil || id != "ITEM_1" {
		t.Errorf("findProjectItemID got %q, %v want ITEM_1", id, err)
	}
	if _, err := findProjectItemID(tc(), "PROJ_OTHER", 123); err == nil {
		t.Error("expected not-found when project id doesn't match")
	}
}

func TestAddProjectItem(t *testing.T) {
	newMock(t)
	id, err := addProjectItem(tc(), "PROJ_1", "ISSUE_1")
	if err != nil || id != "ITEM_1" {
		t.Errorf("addProjectItem got %q, %v want ITEM_1", id, err)
	}
}

func TestAddToBoardAndSet(t *testing.T) {
	newMock(t)
	bf := boardFields{
		Status:    "Backlog",
		Points:    5,
		Iteration: "Sprint 46",
		Fields:    []string{"Component"},
		Values:    []string{"TECH"},
	}
	if !bf.any() {
		t.Fatal("boardFields.any() should be true")
	}
	if err := addToBoardAndSet(tc(), 3, 123, bf); err != nil {
		t.Errorf("addToBoardAndSet: %v", err)
	}
}

func TestResolveMilestoneNumber(t *testing.T) {
	newMock(t)
	if n, err := resolveMilestoneNumber(tc(), "v2.1"); err != nil || n != 7 {
		t.Errorf("open milestone: got %d, %v want 7", n, err)
	}
	if n, err := resolveMilestoneNumber(tc(), "Closed Epic"); err != nil || n != 8 {
		t.Errorf("closed milestone: got %d, %v want 8", n, err)
	}
	if _, err := resolveMilestoneNumber(tc(), "nope"); err == nil {
		t.Error("expected not-found for unknown milestone")
	}
}

func TestFetchFilteredIssues(t *testing.T) {
	newMock(t)
	bulkLabel = []string{"type:bug"}
	bulkMilestone = ""
	defer func() { bulkLabel = nil }()
	nums, err := fetchFilteredIssues(tc())
	if err != nil {
		t.Fatalf("fetchFilteredIssues: %v", err)
	}
	if len(nums) != 2 || nums[0] != 11 || nums[1] != 12 {
		t.Errorf("got %v want [11 12]", nums)
	}
}

func TestWarnAssigneeCap(t *testing.T) {
	many := make([]string, 11)
	for i := range many {
		many[i] = "u"
	}
	if out := captureStderr(func() { warnAssigneeCap(many) }); !strings.Contains(out, "caps issues at 10") {
		t.Errorf("expected cap warning, got %q", out)
	}
	if out := captureStderr(func() { warnAssigneeCap([]string{"a", "b"}) }); out != "" {
		t.Errorf("expected no warning for 2 assignees, got %q", out)
	}
}

func TestWarnDroppedAssignees(t *testing.T) {
	data := json.RawMessage(`{"assignees":[{"login":"alice"}]}`)
	out := captureStderr(func() { warnDroppedAssignees(data, []string{"alice", "ghost"}) })
	if !strings.Contains(out, "ghost") || !strings.Contains(out, "not applied") {
		t.Errorf("expected dropped warning for ghost, got %q", out)
	}
	out = captureStderr(func() { warnDroppedAssignees(data, []string{"alice"}) })
	if out != "" {
		t.Errorf("expected no warning when all applied, got %q", out)
	}
}

func TestFlattenIssues(t *testing.T) {
	data := json.RawMessage(`[{"number":1,"title":"a","state":"open"},{"number":2,"title":"b","state":"closed"}]`)
	out := flattenIssues(data)
	var rows []map[string]any
	if err := json.Unmarshal(out, &rows); err != nil {
		t.Fatalf("flattenIssues output not array: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows want 2", len(rows))
	}
}

// ensure context import is used (helpers above call client methods that take ctx internally)
var _ = context.Background
