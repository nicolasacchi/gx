package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/nicolasacchi/gx/internal/client"
	"github.com/spf13/cobra"
)

var (
	itemsProjectNum int
	itemsStatus     string
	itemsPriority   string
	itemsPoints     float64
	itemsIteration  string
	itemsField      string
	itemsValue      string
	itemsClear      bool
)

var itemsAddProjectNum int

func init() {
	rootCmd.AddCommand(itemsCmd)
	itemsCmd.AddCommand(itemsAddCmd)
	itemsCmd.AddCommand(itemsSetCmd)
	itemsCmd.AddCommand(itemsClearCmd)
	itemsCmd.AddCommand(itemsArchiveCmd)

	itemsAddCmd.Flags().IntVar(&itemsAddProjectNum, "project-number", 0, "Project number (required)")
	itemsAddCmd.MarkFlagRequired("project-number")

	itemsSetCmd.Flags().IntVar(&itemsProjectNum, "project-number", 0, "Project number (required)")
	itemsSetCmd.Flags().StringVar(&itemsStatus, "status", "", "Set status (e.g., 'In Progress')")
	itemsSetCmd.Flags().StringVar(&itemsPriority, "priority", "", "Set priority (e.g., 'High')")
	itemsSetCmd.Flags().Float64Var(&itemsPoints, "points", 0, "Set story points")
	itemsSetCmd.Flags().StringVar(&itemsIteration, "iteration", "", "Set iteration by title")
	itemsSetCmd.Flags().StringVar(&itemsField, "field", "", "Set custom field by name")
	itemsSetCmd.Flags().StringVar(&itemsValue, "value", "", "Value for --field")
	itemsSetCmd.MarkFlagRequired("project-number")

	itemsClearCmd.Flags().IntVar(&itemsProjectNum, "project-number", 0, "Project number (required)")
	itemsClearCmd.Flags().StringVar(&itemsField, "field", "", "Field name to clear (required)")
	itemsClearCmd.MarkFlagRequired("project-number")
	itemsClearCmd.MarkFlagRequired("field")
}

var itemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Set project board fields on issues (auto-resolves IDs)",
}

var itemsAddCmd = &cobra.Command{
	Use:   "add <issue-number>",
	Short: "Add an issue to a project board",
	Long: `Add an existing issue to a GitHub Project board.

Examples:
  gx items add 123 --project-number 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		issueNum, err := parseNumber(args[0])
		if err != nil {
			return err
		}

		projectID, err := c.ProjectNodeID(context.Background(), itemsAddProjectNum)
		if err != nil {
			return err
		}
		issueID, err := c.IssueNodeID(context.Background(), issueNum)
		if err != nil {
			return err
		}

		query := fmt.Sprintf(`mutation {
			addProjectV2ItemById(input: {projectId: %q, contentId: %q}) {
				item { id }
			}
		}`, projectID, issueID)

		data, err := c.GraphQL(context.Background(), query, nil)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "added #%d to project %d\n", issueNum, itemsAddProjectNum)
		}
		// Only emit the mutation JSON to stdout if the caller explicitly asked for it
		// via --json or --jq. Piped-stdout auto-JSON is not enough — scripts loop over
		// many add calls and don't want every success polluting their log stream.
		if jsonFlag || jqFlag != "" {
			return printData("", data)
		}
		return nil
	},
}

var itemsSetCmd = &cobra.Command{
	Use:   "set <issue-number>",
	Short: "Set field values on a project item",
	Long: `Set one or more field values on an issue in a project board.
Auto-resolves field names to IDs and option names to option IDs.

Examples:
  gx items set 123 --project-number 1 --status "In Progress"
  gx items set 123 --project-number 1 --priority "High" --points 5
  gx items set 123 --project-number 1 --iteration "Sprint 46"
  gx items set 123 --project-number 1 --field "Component" --value "TECH"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		issueNum, err := parseNumber(args[0])
		if err != nil {
			return err
		}

		// Get project ID and item ID
		projectID, err := c.ProjectNodeID(context.Background(), itemsProjectNum)
		if err != nil {
			return err
		}

		itemID, err := findProjectItemID(c, projectID, issueNum)
		if err != nil {
			return fmt.Errorf("issue #%d not found in project %d: %w", issueNum, itemsProjectNum, err)
		}

		// Get all fields for auto-resolution
		fields, err := getProjectFields(c, itemsProjectNum)
		if err != nil {
			return err
		}

		updated := 0

		if itemsStatus != "" {
			if err := setFieldValue(c, projectID, itemID, fields, "Status", itemsStatus); err != nil {
				return fmt.Errorf("set status: %w", err)
			}
			updated++
		}
		if itemsPriority != "" {
			if err := setFieldValue(c, projectID, itemID, fields, "Priority", itemsPriority); err != nil {
				return fmt.Errorf("set priority: %w", err)
			}
			updated++
		}
		if itemsPoints > 0 {
			if err := setNumberField(c, projectID, itemID, fields, "Story Points", itemsPoints); err != nil {
				return fmt.Errorf("set points: %w", err)
			}
			updated++
		}
		if itemsIteration != "" {
			if err := setIterationField(c, projectID, itemID, fields, itemsIteration); err != nil {
				return fmt.Errorf("set iteration: %w", err)
			}
			updated++
		}
		if itemsField != "" && itemsValue != "" {
			// Try as single-select first, then text, then number
			if err := setFieldValue(c, projectID, itemID, fields, itemsField, itemsValue); err != nil {
				// Try as number
				if num, parseErr := strconv.ParseFloat(itemsValue, 64); parseErr == nil {
					if err := setNumberField(c, projectID, itemID, fields, itemsField, num); err != nil {
						return fmt.Errorf("set %s: %w", itemsField, err)
					}
				} else {
					// Try as text
					if err := setTextField(c, projectID, itemID, fields, itemsField, itemsValue); err != nil {
						return fmt.Errorf("set %s: %w", itemsField, err)
					}
				}
			}
			updated++
		}

		if updated == 0 {
			return fmt.Errorf("no fields to update; use --status, --priority, --points, --iteration, or --field/--value")
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "updated %d field(s) on #%d\n", updated, issueNum)
		}
		return nil
	},
}

var itemsClearCmd = &cobra.Command{
	Use:   "clear <issue-number>",
	Short: "Clear a field value on a project item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		issueNum, _ := parseNumber(args[0])
		projectID, err := c.ProjectNodeID(context.Background(), itemsProjectNum)
		if err != nil {
			return err
		}
		itemID, err := findProjectItemID(c, projectID, issueNum)
		if err != nil {
			return err
		}
		fields, err := getProjectFields(c, itemsProjectNum)
		if err != nil {
			return err
		}
		fieldID := resolveFieldID(fields, itemsField)
		if fieldID == "" {
			return fmt.Errorf("field %q not found", itemsField)
		}

		query := fmt.Sprintf(`mutation {
			clearProjectV2ItemFieldValue(input: {projectId: %q, itemId: %q, fieldId: %q}) {
				projectV2Item { id }
			}
		}`, projectID, itemID, fieldID)
		if _, err := c.GraphQL(context.Background(), query, nil); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "cleared %s on #%d\n", itemsField, issueNum)
		}
		return nil
	},
}

var itemsArchiveCmd = &cobra.Command{
	Use:   "archive <issue-number>",
	Short: "Archive a project item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		issueNum, _ := parseNumber(args[0])
		projectID, err := c.ProjectNodeID(context.Background(), itemsProjectNum)
		if err != nil {
			return err
		}
		itemID, err := findProjectItemID(c, projectID, issueNum)
		if err != nil {
			return err
		}
		query := fmt.Sprintf(`mutation {
			archiveProjectV2Item(input: {projectId: %q, itemId: %q}) {
				item { id }
			}
		}`, projectID, itemID)
		if _, err := c.GraphQL(context.Background(), query, nil); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "archived #%d from project\n", issueNum)
		}
		return nil
	},
}

// --- helper types and functions ---

type fieldOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type fieldIteration struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type projectField struct {
	ID         string
	Name       string
	DataType   string
	Options    []fieldOption
	Iterations []fieldIteration
}

func getProjectFields(c *client.Client, projectNum int) ([]projectField, error) {
	query := fmt.Sprintf(`{
		organization(login: %q) {
			projectV2(number: %d) {
				fields(first: 50) {
					nodes {
						... on ProjectV2Field { id name dataType }
						... on ProjectV2SingleSelectField { id name dataType options { id name } }
						... on ProjectV2IterationField { id name dataType configuration { iterations { id title } } }
					}
				}
			}
		}
	}`, c.Owner(), projectNum)

	data, err := c.GraphQL(context.Background(), query, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Organization struct {
			ProjectV2 struct {
				Fields struct {
					Nodes []json.RawMessage `json:"nodes"`
				} `json:"fields"`
			} `json:"projectV2"`
		} `json:"organization"`
	}
	json.Unmarshal(data, &resp)

	var fields []projectField
	for _, raw := range resp.Organization.ProjectV2.Fields.Nodes {
		var f struct {
			ID            string           `json:"id"`
			Name          string           `json:"name"`
			DataType      string           `json:"dataType"`
			Options       []fieldOption    `json:"options"`
			Configuration *struct {
				Iterations []fieldIteration `json:"iterations"`
			} `json:"configuration"`
		}
		json.Unmarshal(raw, &f)
		if f.ID == "" {
			continue
		}
		pf := projectField{
			ID: f.ID, Name: f.Name, DataType: f.DataType,
			Options: f.Options,
		}
		if f.Configuration != nil {
			pf.Iterations = f.Configuration.Iterations
		}
		fields = append(fields, pf)
	}
	return fields, nil
}

func resolveFieldID(fields []projectField, name string) string {
	for _, f := range fields {
		if equalsIgnoreCase(f.Name, name) {
			return f.ID
		}
	}
	return ""
}

func resolveOptionID(fields []projectField, fieldName, optionName string) (string, string) {
	for _, f := range fields {
		if !equalsIgnoreCase(f.Name, fieldName) {
			continue
		}
		for _, o := range f.Options {
			if equalsIgnoreCase(o.Name, optionName) {
				return f.ID, o.ID
			}
		}
	}
	return "", ""
}

func resolveIterationID(fields []projectField, iterTitle string) (string, string) {
	for _, f := range fields {
		for _, it := range f.Iterations {
			if equalsIgnoreCase(it.Title, iterTitle) {
				return f.ID, it.ID
			}
		}
	}
	return "", ""
}

func equalsIgnoreCase(a, b string) bool {
	return len(a) == len(b) && (a == b || lower(a) == lower(b))
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

func findProjectItemID(c *client.Client, projectID string, issueNum int) (string, error) {
	issueID, err := c.IssueNodeID(context.Background(), issueNum)
	if err != nil {
		return "", err
	}

	// Query the issue's projectItems directly — O(1) regardless of project size.
	// This avoids the old approach of scanning all project items (which broke at >100).
	query := fmt.Sprintf(`{
		node(id: %q) {
			... on Issue {
				projectItems(first: 50) {
					nodes {
						id
						project { id }
					}
				}
			}
		}
	}`, issueID)

	data, err := c.GraphQL(context.Background(), query, nil)
	if err != nil {
		return "", err
	}

	var resp struct {
		Node struct {
			ProjectItems struct {
				Nodes []struct {
					ID      string `json:"id"`
					Project struct {
						ID string `json:"id"`
					} `json:"project"`
				} `json:"nodes"`
			} `json:"projectItems"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse project items: %w", err)
	}

	for _, item := range resp.Node.ProjectItems.Nodes {
		if item.Project.ID == projectID {
			return item.ID, nil
		}
	}
	return "", fmt.Errorf("issue not found in project")
}

func setFieldValue(c *client.Client, projectID, itemID string, fields []projectField, fieldName, value string) error {
	fieldID, optionID := resolveOptionID(fields, fieldName, value)
	if fieldID == "" || optionID == "" {
		return fmt.Errorf("field %q option %q not found", fieldName, value)
	}
	query := fmt.Sprintf(`mutation {
		updateProjectV2ItemFieldValue(input: {
			projectId: %q, itemId: %q, fieldId: %q,
			value: {singleSelectOptionId: %q}
		}) { projectV2Item { id } }
	}`, projectID, itemID, fieldID, optionID)
	_, err := c.GraphQL(context.Background(), query, nil)
	return err
}

func setNumberField(c *client.Client, projectID, itemID string, fields []projectField, fieldName string, value float64) error {
	fieldID := resolveFieldID(fields, fieldName)
	if fieldID == "" {
		return fmt.Errorf("field %q not found", fieldName)
	}
	query := fmt.Sprintf(`mutation {
		updateProjectV2ItemFieldValue(input: {
			projectId: %q, itemId: %q, fieldId: %q,
			value: {number: %f}
		}) { projectV2Item { id } }
	}`, projectID, itemID, fieldID, value)
	_, err := c.GraphQL(context.Background(), query, nil)
	return err
}

func setTextField(c *client.Client, projectID, itemID string, fields []projectField, fieldName, value string) error {
	fieldID := resolveFieldID(fields, fieldName)
	if fieldID == "" {
		return fmt.Errorf("field %q not found", fieldName)
	}
	query := fmt.Sprintf(`mutation {
		updateProjectV2ItemFieldValue(input: {
			projectId: %q, itemId: %q, fieldId: %q,
			value: {text: %q}
		}) { projectV2Item { id } }
	}`, projectID, itemID, fieldID, value)
	_, err := c.GraphQL(context.Background(), query, nil)
	return err
}

func setIterationField(c *client.Client, projectID, itemID string, fields []projectField, iterTitle string) error {
	fieldID, iterID := resolveIterationID(fields, iterTitle)
	if fieldID == "" || iterID == "" {
		return fmt.Errorf("iteration %q not found", iterTitle)
	}
	query := fmt.Sprintf(`mutation {
		updateProjectV2ItemFieldValue(input: {
			projectId: %q, itemId: %q, fieldId: %q,
			value: {iterationId: %q}
		}) { projectV2Item { id } }
	}`, projectID, itemID, fieldID, iterID)
	_, err := c.GraphQL(context.Background(), query, nil)
	return err
}
