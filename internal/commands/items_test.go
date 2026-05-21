package commands

import (
	"testing"

	"github.com/nicolasacchi/gx/internal/client"
)

func testFields() []projectField {
	return []projectField{
		{ID: "f1", Name: "Status", DataType: "SINGLE_SELECT", Options: []fieldOption{{ID: "o1", Name: "Backlog"}, {ID: "o2", Name: "Done"}}},
		{ID: "f2", Name: "Story Points", DataType: "NUMBER"},
		{ID: "f3", Name: "Jira Key", DataType: "TEXT"},
		{ID: "f4", Name: "Target date", DataType: "DATE"},
		{ID: "f5", Name: "Sprint", DataType: "ITERATION", Iterations: []fieldIteration{{ID: "i1", Title: "Sprint 46"}}},
	}
}

func TestEqualsIgnoreCase(t *testing.T) {
	if !equalsIgnoreCase("Status", "status") {
		t.Error("Status/status should match")
	}
	if equalsIgnoreCase("Status", "Priority") {
		t.Error("Status/Priority should not match")
	}
}

func TestResolveFieldID(t *testing.T) {
	f := testFields()
	if resolveFieldID(f, "story points") != "f2" {
		t.Error("case-insensitive field lookup failed")
	}
	if resolveFieldID(f, "nope") != "" {
		t.Error("absent field should return empty")
	}
}

func TestResolveOptionID(t *testing.T) {
	fid, oid := resolveOptionID(testFields(), "Status", "done")
	if fid != "f1" || oid != "o2" {
		t.Errorf("got %s/%s want f1/o2", fid, oid)
	}
	if _, oid := resolveOptionID(testFields(), "Status", "nope"); oid != "" {
		t.Error("absent option should return empty option id")
	}
}

func TestResolveIterationID(t *testing.T) {
	fid, iid := resolveIterationID(testFields(), "sprint 46")
	if fid != "f5" || iid != "i1" {
		t.Errorf("got %s/%s want f5/i1", fid, iid)
	}
}

func TestValidDate(t *testing.T) {
	if !validDate("2026-06-01") {
		t.Error("YYYY-MM-DD should be valid")
	}
	if validDate("2026-13-99") {
		t.Error("impossible month/day should be invalid")
	}
	if validDate("June 1") {
		t.Error("wrong format should be invalid")
	}
}

// TestSetCustomFieldErrorBranches covers the paths that return before any
// network call — type-dispatched validation errors.
func TestSetCustomFieldErrorBranches(t *testing.T) {
	c := client.New("t", "o", "r", false)
	f := testFields()
	cases := []struct {
		name, field, value string
	}{
		{"unknown field", "Nonexistent", "x"},
		{"bad single-select option", "Status", "NotAnOption"},
		{"non-numeric for number field", "Story Points", "notanumber"},
		{"bad date for date field", "Target date", "nope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := setCustomField(c, "proj", "item", f, tc.field, tc.value); err == nil {
				t.Errorf("expected error for %s/%s", tc.field, tc.value)
			}
		})
	}
}

func TestSetCustomFieldsLengthMismatch(t *testing.T) {
	c := client.New("t", "o", "r", false)
	if _, err := setCustomFields(c, "p", "i", testFields(), []string{"Status"}, []string{}); err == nil {
		t.Error("expected error when --field and --value counts differ")
	}
	if n, err := setCustomFields(c, "p", "i", testFields(), nil, nil); err != nil || n != 0 {
		t.Errorf("empty should be no-op: n=%d err=%v", n, err)
	}
}
