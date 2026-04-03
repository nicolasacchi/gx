package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
)

type FormatFunc func(any) string

type ColumnDef struct {
	Header string
	Key    string
	Format FormatFunc
}

var commandColumns = map[string][]ColumnDef{
	"issues.list": {
		{Header: "#", Key: "number"},
		{Header: "STATE", Key: "state"},
		{Header: "MILESTONE", Key: "milestone"},
		{Header: "ASSIGNEE", Key: "assignee"},
		{Header: "TITLE", Key: "title", Format: truncate60},
	},
	"sub-issues.list": {
		{Header: "#", Key: "number"},
		{Header: "STATE", Key: "state"},
		{Header: "TITLE", Key: "title", Format: truncate60},
	},
	"milestones.list": {
		{Header: "#", Key: "number"},
		{Header: "TITLE", Key: "title"},
		{Header: "STATE", Key: "state"},
		{Header: "OPEN", Key: "open_issues"},
		{Header: "CLOSED", Key: "closed_issues"},
		{Header: "DUE", Key: "due_on"},
	},
	"iterations.list": {
		{Header: "ID", Key: "id"},
		{Header: "TITLE", Key: "title"},
		{Header: "START", Key: "start_date"},
		{Header: "DURATION", Key: "duration"},
	},
	"board.list": {
		{Header: "#", Key: "number"},
		{Header: "TITLE", Key: "title"},
		{Header: "STATE", Key: "state"},
	},
	"board.fields": {
		{Header: "ID", Key: "id"},
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
		{Header: "OPTIONS", Key: "options", Format: truncate60},
	},
	"items.list": {
		{Header: "#", Key: "number"},
		{Header: "STATUS", Key: "status"},
		{Header: "PRIORITY", Key: "priority"},
		{Header: "TITLE", Key: "title", Format: truncate60},
	},
	"comments.list": {
		{Header: "ID", Key: "id"},
		{Header: "USER", Key: "user"},
		{Header: "CREATED", Key: "created_at"},
		{Header: "BODY", Key: "body", Format: truncate60},
	},
	"labels.list": {
		{Header: "NAME", Key: "name"},
		{Header: "COLOR", Key: "color"},
		{Header: "DESCRIPTION", Key: "description"},
	},
	"search": {
		{Header: "#", Key: "number"},
		{Header: "STATE", Key: "state"},
		{Header: "TITLE", Key: "title", Format: truncate60},
	},
	"config.list": {
		{Header: "NAME", Key: "name"},
		{Header: "OWNER", Key: "owner"},
		{Header: "REPO", Key: "repo"},
		{Header: "DEFAULT", Key: "default"},
	},
}

func printTable(command string, data json.RawMessage) error {
	columns, ok := commandColumns[command]
	if !ok {
		return fmt.Errorf("no table definition for %s", command)
	}

	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		var single map[string]any
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return fmt.Errorf("cannot render as table")
		}
		rows = []map[string]any{single}
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if term.IsTerminal(int(os.Stdout.Fd())) {
		t.SetStyle(table.StyleLight)
	} else {
		t.SetStyle(table.StyleDefault)
	}

	header := make(table.Row, len(columns))
	for i, col := range columns {
		header[i] = col.Header
	}
	t.AppendHeader(header)

	for _, row := range rows {
		r := make(table.Row, len(columns))
		for i, col := range columns {
			if col.Format != nil {
				r[i] = col.Format(row[col.Key])
			} else {
				r[i] = formatValue(row[col.Key])
			}
		}
		t.AppendRow(r)
	}

	t.Render()
	return nil
}

func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case []any:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(parts, ", ")
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func truncate60(v any) string {
	s := formatValue(v)
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}
