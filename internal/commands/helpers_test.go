package commands

import (
	"encoding/json"
	"testing"
)

func TestFlattenIssue(t *testing.T) {
	raw := json.RawMessage(`{
		"number":7,"title":"T","state":"open","body":"B","html_url":"u",
		"assignees":[{"login":"a"},{"login":"b"}],
		"labels":[{"name":"bug"}],
		"milestone":{"number":3,"title":"M"},
		"type":{"name":"Task"}
	}`)
	flat := flattenIssue(raw)
	if flat == nil {
		t.Fatal("got nil")
	}
	if flat["number"] != 7 {
		t.Errorf("number=%v", flat["number"])
	}
	if flat["type"] != "Task" {
		t.Errorf("type=%v want Task", flat["type"])
	}
	if flat["milestone"] != "M" || flat["milestone_number"] != 3 {
		t.Errorf("milestone=%v/%v", flat["milestone"], flat["milestone_number"])
	}
	as, ok := flat["assignees"].([]string)
	if !ok || len(as) != 2 || as[0] != "a" || as[1] != "b" {
		t.Errorf("assignees=%v", flat["assignees"])
	}
	ls, ok := flat["labels"].([]string)
	if !ok || len(ls) != 1 || ls[0] != "bug" {
		t.Errorf("labels=%v", flat["labels"])
	}
}

func TestFlattenIssueTypeAbsentWhenMissing(t *testing.T) {
	flat := flattenIssue(json.RawMessage(`{"number":1,"title":"x","state":"open"}`))
	if flat == nil {
		t.Fatal("got nil")
	}
	if _, ok := flat["type"]; ok {
		t.Error("type key should be absent when the issue has no type")
	}
}

func TestFlattenIssueBadJSON(t *testing.T) {
	if flattenIssue(json.RawMessage(`not json`)) != nil {
		t.Error("want nil on invalid JSON")
	}
}

func TestParseNumber(t *testing.T) {
	if n, err := parseNumber("42"); err != nil || n != 42 {
		t.Errorf("parseNumber(42)=%d,%v", n, err)
	}
	if _, err := parseNumber("abc"); err == nil {
		t.Error("parseNumber(abc) want error")
	}
}
