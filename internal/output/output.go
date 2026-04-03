package output

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/term"
)

// IsJSON returns true if output should be JSON: --json flag, --jq filter, or piped stdout.
func IsJSON(jsonFlag bool, jqFilter string) bool {
	if jsonFlag || jqFilter != "" {
		return true
	}
	return !term.IsTerminal(int(os.Stdout.Fd()))
}

// PrintData dispatches to JSON or table output based on mode.
func PrintData(command string, data json.RawMessage, isJSON bool, jqFilter string) error {
	if isJSON {
		return printJSON(data, jqFilter)
	}
	if err := printTable(command, data); err != nil {
		return printJSON(data, "")
	}
	return nil
}

// PrintError prints an error to stderr.
func PrintError(msg string, statusCode int) {
	if statusCode > 0 {
		fmt.Fprintf(os.Stderr, "error: %d — %s\n", statusCode, msg)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", msg)
	}
}

func printJSON(data json.RawMessage, jqFilter string) error {
	if jqFilter != "" {
		filtered, err := ApplyFilter(data, jqFilter)
		if err != nil {
			return err
		}
		data = filtered
	}
	out, err := json.MarshalIndent(json.RawMessage(data), "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stdout, string(data))
		return nil
	}
	fmt.Fprintln(os.Stdout, string(out))
	return nil
}
