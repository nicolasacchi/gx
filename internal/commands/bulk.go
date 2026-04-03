package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/nicolasacchi/gx/internal/client"
	"github.com/spf13/cobra"
)

var (
	bulkLabel     []string
	bulkMilestone string
	bulkState     string
	bulkAddLabel  string
	bulkReason    string
)

func init() {
	rootCmd.AddCommand(bulkCmd)
	bulkCmd.AddCommand(bulkEditCmd)
	bulkCmd.AddCommand(bulkCloseCmd)

	bulkEditCmd.Flags().StringSliceVar(&bulkLabel, "label", nil, "Filter by labels")
	bulkEditCmd.Flags().StringVar(&bulkMilestone, "milestone", "", "Filter by milestone")
	bulkEditCmd.Flags().StringVar(&bulkAddLabel, "add-label", "", "Label to add to all matching issues")

	bulkCloseCmd.Flags().StringSliceVar(&bulkLabel, "label", nil, "Filter by labels")
	bulkCloseCmd.Flags().StringVar(&bulkMilestone, "milestone", "", "Filter by milestone")
	bulkCloseCmd.Flags().StringVar(&bulkReason, "reason", "", "Close reason: completed, not_planned")
}

var bulkCmd = &cobra.Command{
	Use:   "bulk",
	Short: "Batch operations on multiple issues",
}

var bulkEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Bulk edit issues matching filters",
	Long: `Add labels to all issues matching a filter.

Examples:
  gx bulk edit --label "type:bug" --add-label "must-do"
  gx bulk edit --milestone "v2.1" --add-label "ready"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if bulkAddLabel == "" {
			return fmt.Errorf("specify --add-label")
		}

		issues, err := fetchFilteredIssues(c)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "bulk edit: %d issues\n", len(issues))
		var success, fail int
		for _, num := range issues {
			_, err := c.Post(context.Background(), fmt.Sprintf("issues/%d/labels", num), map[string]any{"labels": []string{bulkAddLabel}})
			if err != nil {
				fmt.Fprintf(os.Stderr, "  failed: #%d — %s\n", num, err)
				fail++
			} else {
				success++
			}
		}
		fmt.Fprintf(os.Stderr, "done: %d updated, %d failed\n", success, fail)
		return nil
	},
}

var bulkCloseCmd = &cobra.Command{
	Use:   "close",
	Short: "Bulk close issues matching filters",
	Long: `Close all issues matching a filter.

Examples:
  gx bulk close --label "sdd:problem" --reason "not_planned"
  gx bulk close --milestone "old-milestone"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		issues, err := fetchFilteredIssues(c)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "bulk close: %d issues\n", len(issues))
		body := map[string]any{"state": "closed"}
		if bulkReason != "" {
			body["state_reason"] = bulkReason
		}

		var success, fail int
		for _, num := range issues {
			_, err := c.Patch(context.Background(), fmt.Sprintf("issues/%d", num), body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  failed: #%d — %s\n", num, err)
				fail++
			} else {
				success++
			}
		}
		fmt.Fprintf(os.Stderr, "done: %d closed, %d failed\n", success, fail)
		return nil
	},
}

func fetchFilteredIssues(c *client.Client) ([]int, error) {
	params := url.Values{
		"state":    {"open"},
		"per_page": {"100"},
	}
	if len(bulkLabel) > 0 {
		params.Set("labels", strings.Join(bulkLabel, ","))
	}
	if bulkMilestone != "" {
		// Resolve milestone title to number
		num, err := resolveMilestoneNumber(c, bulkMilestone)
		if err != nil {
			return nil, err
		}
		params.Set("milestone", fmt.Sprintf("%d", num))
	}

	data, err := c.Get(context.Background(), "issues", params)
	if err != nil {
		return nil, err
	}

	var issues []struct {
		Number int `json:"number"`
	}
	json.Unmarshal(data, &issues)

	nums := make([]int, len(issues))
	for i, issue := range issues {
		nums[i] = issue.Number
	}
	return nums, nil
}
