package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/spf13/cobra"
)

var overviewProjectNum int

func init() {
	rootCmd.AddCommand(overviewCmd)
	overviewCmd.Flags().IntVar(&overviewProjectNum, "project-number", 0, "Project number (optional)")
}

var overviewCmd = &cobra.Command{
	Use:   "overview",
	Short: "Project health snapshot (parallel fetch)",
	Long: `Parallel fetch of: open issues by label, milestones progress,
blocked items, must-do backlog count.

Examples:
  gx overview
  gx overview --project-number 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		result := map[string]any{"owner": c.Owner(), "repo": c.Repo()}
		var mu sync.Mutex
		var wg sync.WaitGroup

		// Open issues count
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := c.Get(context.Background(), "issues", url.Values{"state": {"open"}, "per_page": {"1"}})
			if err != nil {
				mu.Lock()
				result["open_issues_error"] = err.Error()
				mu.Unlock()
				return
			}
			var issues []json.RawMessage
			json.Unmarshal(data, &issues)
			mu.Lock()
			result["open_issues_sample"] = len(issues)
			mu.Unlock()
		}()

		// Must-do count
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := c.Get(context.Background(), "issues", url.Values{"state": {"open"}, "labels": {"must-do"}, "per_page": {"100"}})
			if err != nil {
				mu.Lock()
				result["must_do_error"] = err.Error()
				mu.Unlock()
				return
			}
			var issues []json.RawMessage
			json.Unmarshal(data, &issues)
			mu.Lock()
			result["must_do_count"] = len(issues)
			mu.Unlock()
		}()

		// Milestones
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := c.Get(context.Background(), "milestones", url.Values{"state": {"open"}, "per_page": {"10"}})
			if err != nil {
				mu.Lock()
				result["milestones_error"] = err.Error()
				mu.Unlock()
				return
			}
			var milestones []struct {
				Title        string `json:"title"`
				OpenIssues   int    `json:"open_issues"`
				ClosedIssues int    `json:"closed_issues"`
				DueOn        string `json:"due_on"`
			}
			json.Unmarshal(data, &milestones)
			var ms []map[string]any
			for _, m := range milestones {
				total := m.OpenIssues + m.ClosedIssues
				pct := 0
				if total > 0 {
					pct = m.ClosedIssues * 100 / total
				}
				ms = append(ms, map[string]any{
					"title":    m.Title,
					"progress": fmt.Sprintf("%d%% (%d/%d)", pct, m.ClosedIssues, total),
					"due":      m.DueOn,
				})
			}
			mu.Lock()
			result["milestones"] = ms
			mu.Unlock()
		}()

		// Bug count
		wg.Add(1)
		go func() {
			defer wg.Done()
			data, err := c.Get(context.Background(), "issues", url.Values{"state": {"open"}, "labels": {"type:bug"}, "per_page": {"100"}})
			if err != nil {
				return
			}
			var issues []json.RawMessage
			json.Unmarshal(data, &issues)
			mu.Lock()
			result["open_bugs"] = len(issues)
			mu.Unlock()
		}()

		wg.Wait()

		out, _ := json.Marshal(result)
		return printData("", out)
	},
}
