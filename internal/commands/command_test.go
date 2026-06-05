package commands

import (
	"strings"
	"testing"
)

// base flags every command needs so getClient resolves without gh/config.
func baseArgs(extra ...string) []string {
	return append([]string{"--token", "tok", "--owner", "1000farmacie", "--repo", "1000farmacie"}, extra...)
}

func TestCmdIssuesCreate(t *testing.T) {
	newMock(t)
	out, err := runGx(t, baseArgs("issues", "create", "--title", "Fix", "--type", "Bug")...)
	if err != nil {
		t.Fatalf("issues create: %v", err)
	}
	if !strings.Contains(out, "14710") || !strings.Contains(out, "Task") {
		t.Errorf("expected created issue JSON, got: %s", out)
	}
}

func TestCmdIssuesCreateWithBoard(t *testing.T) {
	newMock(t)
	_, err := runGx(t, baseArgs("issues", "create", "--title", "Full", "--type", "Task",
		"--project-number", "3", "--status", "Backlog", "--field", "Component", "--value", "TECH")...)
	if err != nil {
		t.Fatalf("issues create + board: %v", err)
	}
}

func TestCmdIssuesEdit(t *testing.T) {
	newMock(t)
	out, err := runGx(t, baseArgs("issues", "edit", "123", "--title", "New", "--type", "Bug")...)
	if err != nil {
		t.Fatalf("issues edit: %v", err)
	}
	if !strings.Contains(out, "123") {
		t.Errorf("expected echoed issue JSON, got: %s", out)
	}
}

func TestCmdItemsSet(t *testing.T) {
	newMock(t)
	if _, err := runGx(t, baseArgs("items", "set", "123", "--project-number", "3", "--status", "Backlog")...); err != nil {
		t.Fatalf("items set: %v", err)
	}
}

func TestCmdItemsGet(t *testing.T) {
	newMock(t)
	out, err := runGx(t, baseArgs("items", "get", "123", "--project-number", "3")...)
	if err != nil {
		t.Fatalf("items get: %v", err)
	}
	if !strings.Contains(out, "Backlog") {
		t.Errorf("expected current field values, got: %s", out)
	}
}

func TestCmdItemsAddDraft(t *testing.T) {
	newMock(t)
	out, err := runGx(t, baseArgs("items", "add-draft", "--project-number", "3", "--title", "Spike")...)
	if err != nil {
		t.Fatalf("items add-draft: %v", err)
	}
	if !strings.Contains(out, "DRAFT_1") {
		t.Errorf("expected draft item id, got: %s", out)
	}
}

func TestCmdCommentsAdd(t *testing.T) {
	newMock(t)
	out, err := runGx(t, baseArgs("comments", "add", "123", "--body", "hi")...)
	if err != nil {
		t.Fatalf("comments add: %v", err)
	}
	if !strings.Contains(out, "555") {
		t.Errorf("expected comment JSON, got: %s", out)
	}
}

func TestCmdIssuesTransfer(t *testing.T) {
	newMock(t)
	if _, err := runGx(t, baseArgs("issues", "transfer", "123", "--to-repo", "other")...); err != nil {
		t.Fatalf("issues transfer: %v", err)
	}
}

func TestCmdIssuesDevelop(t *testing.T) {
	newMock(t)
	if _, err := runGx(t, baseArgs("issues", "develop", "123")...); err != nil {
		t.Fatalf("issues develop: %v", err)
	}
}

func TestCmdBulkClose(t *testing.T) {
	newMock(t)
	if _, err := runGx(t, baseArgs("bulk", "close", "--label", "type:bug", "--yes")...); err != nil {
		t.Fatalf("bulk close: %v", err)
	}
}

// TestCmdReadAndSimple drives the remaining read/simple commands through the
// mock to cover their RunE handlers (output not asserted beyond no-error).
func TestCmdReadAndSimple(t *testing.T) {
	cases := [][]string{
		{"issues", "list", "--label", "type:bug"},
		{"issues", "get", "123"},
		{"issues", "close", "123", "--reason", "completed"},
		{"issues", "reopen", "123"},
		{"issues", "edit", "123", "--add-label", "x", "--remove-label", "y"},
		{"issues", "edit", "123", "--milestone", "v2.1"},
		{"items", "clear", "123", "--project-number", "3", "--field", "Story Points"},
		{"items", "archive", "123", "--project-number", "3"},
		{"items", "convert-draft", "DRAFT_1", "--project-number", "3"},
		{"items", "add", "123", "--project-number", "3"},
		{"labels", "list"},
		{"milestones", "list"},
		{"comments", "list", "123"},
		{"comments", "edit", "555", "--body", "edited"},
		{"comments", "delete", "555", "--yes"},
		{"board", "fields", "--project-number", "3"},
		{"sub-issues", "add", "123", "456"},
		{"sub-issues", "remove", "123", "456"},
		{"sub-issues", "reorder", "123", "456", "--after", "789"},
		{"issues", "assign", "123", "--user", "alice"},
		{"issues", "pin", "123"},
		{"issues", "unpin", "123"},
		{"issues", "lock", "123"},
		{"issues", "unlock", "123"},
		{"issues", "timeline", "123"},
		{"issues", "linked-prs", "123"},
		{"milestones", "create", "--title", "v3", "--due", "2026-06-01"},
		{"milestones", "close", "1"},
		{"iterations", "list", "--project-number", "3"},
		{"iterations", "current", "--project-number", "3"},
		{"board", "list"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			newMock(t)
			if _, err := runGx(t, baseArgs(args...)...); err != nil {
				t.Errorf("%v: %v", args, err)
			}
		})
	}
}
