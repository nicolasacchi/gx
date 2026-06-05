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
	bulkLabel        []string
	bulkMilestone    string
	bulkState        string
	bulkAddLabels    []string
	bulkRemLabels    []string
	bulkSetMilestone string
	bulkReason       string
)

func init() {
	rootCmd.AddCommand(bulkCmd)
	bulkCmd.AddCommand(bulkEditCmd)
	bulkCmd.AddCommand(bulkCloseCmd)

	bulkEditCmd.Flags().StringSliceVar(&bulkLabel, "label", nil, "Filter by labels")
	bulkEditCmd.Flags().StringVar(&bulkMilestone, "milestone", "", "Filter by milestone")
	bulkEditCmd.Flags().StringSliceVar(&bulkAddLabels, "add-label", nil, "Label(s) to add to all matching issues")
	bulkEditCmd.Flags().StringSliceVar(&bulkRemLabels, "remove-label", nil, "Label(s) to remove from all matching issues")
	bulkEditCmd.Flags().StringVar(&bulkSetMilestone, "set-milestone", "", "Set milestone (by title) on all matching issues")

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
	Long: `Add/remove labels or set a milestone on all issues matching a filter.

Examples:
  gx bulk edit --label "type:bug" --add-label "must-do"
  gx bulk edit --milestone "v2.1" --add-label "ready" --remove-label "stale"
  gx bulk edit --label "sdd:problem" --set-milestone "Triage"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if len(bulkAddLabels) == 0 && len(bulkRemLabels) == 0 && bulkSetMilestone == "" {
			return fmt.Errorf("specify at least one of --add-label, --remove-label, --set-milestone")
		}
		// Refuse an unscoped bulk edit: an empty filter resolves to the first 100
		// open issues, so a stray invocation would rewrite live tickets en masse.
		if len(bulkLabel) == 0 && bulkMilestone == "" {
			return fmt.Errorf("bulk edit requires at least one selector: --label or --milestone")
		}

		var milestoneNum int
		if bulkSetMilestone != "" {
			milestoneNum, err = resolveMilestoneNumber(c, bulkSetMilestone)
			if err != nil {
				return err
			}
		}

		issues, err := fetchFilteredIssues(c)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "bulk edit: %d issues match (labels=%v milestone=%q)\n", len(issues), bulkLabel, bulkMilestone)
		if dryRun() {
			fmt.Fprintf(os.Stderr, "--dry-run: would edit %d issues, no changes made\n", len(issues))
			return nil
		}
		if err := requireConfirm(fmt.Sprintf("editing %d issues", len(issues))); err != nil {
			return err
		}
		var success, fail int
		for _, num := range issues {
			ok := true
			for _, l := range bulkAddLabels {
				if _, e := c.Post(context.Background(), fmt.Sprintf("issues/%d/labels", num), map[string]any{"labels": []string{l}}); e != nil {
					fmt.Fprintf(os.Stderr, "  failed: #%d add %q — %s\n", num, l, e)
					ok = false
				}
			}
			for _, l := range bulkRemLabels {
				if e := c.Delete(context.Background(), fmt.Sprintf("issues/%d/labels/%s", num, url.PathEscape(l))); e != nil {
					fmt.Fprintf(os.Stderr, "  failed: #%d remove %q — %s\n", num, l, e)
					ok = false
				}
			}
			if bulkSetMilestone != "" {
				if _, e := c.Patch(context.Background(), fmt.Sprintf("issues/%d", num), map[string]any{"milestone": milestoneNum}); e != nil {
					fmt.Fprintf(os.Stderr, "  failed: #%d set-milestone — %s\n", num, e)
					ok = false
				}
			}
			if ok {
				success++
			} else {
				fail++
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
		// Refuse an unscoped bulk close: with no filter, fetchFilteredIssues
		// resolves to the first 100 open issues, so a stray `gx bulk close`
		// would silently close up to 100 live tickets on Project #3.
		if len(bulkLabel) == 0 && bulkMilestone == "" {
			return fmt.Errorf("bulk close requires at least one selector: --label or --milestone")
		}

		issues, err := fetchFilteredIssues(c)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "bulk close: %d issues match (labels=%v milestone=%q)\n", len(issues), bulkLabel, bulkMilestone)
		if dryRun() {
			fmt.Fprintf(os.Stderr, "--dry-run: would close %d issues, no changes made\n", len(issues))
			return nil
		}
		if err := requireConfirm(fmt.Sprintf("closing %d issues", len(issues))); err != nil {
			return err
		}
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
