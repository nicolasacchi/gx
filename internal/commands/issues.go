package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/nicolasacchi/gx/internal/client"
	"github.com/spf13/cobra"
)

var (
	issueState     string
	issueLabel     []string
	issueMilestone string
	issueAssignee  string
	issueTitle     string
	issueBody      string
	issueBodyFile  string
	issueParent    int
	issueAddLabel  []string
	issueRemLabel  []string
	issueCloseReason string
	issueUser      string
)

func init() {
	rootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesGetCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesEditCmd)
	issuesCmd.AddCommand(issuesCloseCmd)
	issuesCmd.AddCommand(issuesReopenCmd)
	issuesCmd.AddCommand(issuesAssignCmd)
	issuesCmd.AddCommand(issuesTimelineCmd)
	issuesCmd.AddCommand(issuesLinkedPRsCmd)
	issuesCmd.AddCommand(issuesPinCmd)
	issuesCmd.AddCommand(issuesUnpinCmd)
	issuesCmd.AddCommand(issuesLockCmd)
	issuesCmd.AddCommand(issuesUnlockCmd)

	issuesListCmd.Flags().StringVar(&issueState, "state", "open", "Filter by state: open, closed, all")
	issuesListCmd.Flags().StringSliceVar(&issueLabel, "label", nil, "Filter by labels")
	issuesListCmd.Flags().StringVar(&issueMilestone, "milestone", "", "Filter by milestone title")
	issuesListCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Filter by assignee")

	issuesCreateCmd.Flags().StringVar(&issueTitle, "title", "", "Issue title (required)")
	issuesCreateCmd.Flags().StringVar(&issueBody, "body", "", "Issue body")
	issuesCreateCmd.Flags().StringVar(&issueBodyFile, "body-file", "", "Read body from file")
	issuesCreateCmd.Flags().StringSliceVar(&issueLabel, "label", nil, "Labels")
	issuesCreateCmd.Flags().StringVar(&issueMilestone, "milestone", "", "Milestone title")
	issuesCreateCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Assignee login")
	issuesCreateCmd.Flags().IntVar(&issueParent, "parent", 0, "Parent issue number (creates as sub-issue)")
	issuesCreateCmd.MarkFlagRequired("title")

	issuesEditCmd.Flags().StringVar(&issueTitle, "title", "", "New title")
	issuesEditCmd.Flags().StringSliceVar(&issueAddLabel, "add-label", nil, "Add labels")
	issuesEditCmd.Flags().StringSliceVar(&issueRemLabel, "remove-label", nil, "Remove labels")

	issuesCloseCmd.Flags().StringVar(&issueCloseReason, "reason", "", "Close reason: completed, not_planned")

	issuesAssignCmd.Flags().StringVar(&issueUser, "user", "", "Assignee login (required)")
	issuesAssignCmd.MarkFlagRequired("user")
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Manage GitHub issues",
}

var issuesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		params := url.Values{
			"state":    {issueState},
			"per_page": {strconv.Itoa(limitFlag)},
			"sort":     {"updated"},
			"direction": {"desc"},
		}
		if len(issueLabel) > 0 {
			params.Set("labels", strings.Join(issueLabel, ","))
		}
		if issueMilestone != "" {
			params.Set("milestone", issueMilestone)
		}
		if issueAssignee != "" {
			params.Set("assignee", issueAssignee)
		}

		data, err := c.Get(context.Background(), "issues", params)
		if err != nil {
			return err
		}

		return printData("issues.list", flattenIssues(data))
	},
}

var issuesGetCmd = &cobra.Command{
	Use:   "get <number>",
	Short: "Get an issue by number",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		data, err := c.Get(context.Background(), "issues/"+args[0], nil)
		if err != nil {
			return err
		}

		if flat := flattenIssue(data); flat != nil {
			out, _ := json.Marshal(flat)
			return printData("", out)
		}
		return printData("", data)
	},
}

var issuesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an issue",
	Long: `Create a GitHub issue. Optionally link as sub-issue with --parent.

Examples:
  gx issues create --title "Fix login bug" --label "type:bug"
  gx issues create --title "Phase 1" --milestone "CoMarketing" --parent 456`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		body := issueBody
		if issueBodyFile != "" {
			content, err := os.ReadFile(issueBodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}
			body = string(content)
		}

		fields := map[string]any{
			"title": issueTitle,
		}
		if body != "" {
			fields["body"] = body
		}
		if len(issueLabel) > 0 {
			fields["labels"] = issueLabel
		}
		if issueAssignee != "" {
			fields["assignees"] = []string{issueAssignee}
		}
		// Milestone requires the milestone number, not title. Look it up.
		if issueMilestone != "" {
			milestoneNum, err := resolveMilestoneNumber(c, issueMilestone)
			if err != nil {
				return err
			}
			fields["milestone"] = milestoneNum
		}

		data, err := c.Post(context.Background(), "issues", fields)
		if err != nil {
			return err
		}

		var created struct {
			Number  int    `json:"number"`
			HTMLURL string `json:"html_url"`
		}
		json.Unmarshal(data, &created)

		// If --parent specified, link as sub-issue via GraphQL
		if issueParent > 0 && created.Number > 0 {
			parentID, err := c.IssueNodeID(context.Background(), issueParent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: issue created (#%d) but failed to link as sub-issue: %s\n", created.Number, err)
			} else {
				childID, err := c.IssueNodeID(context.Background(), created.Number)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: issue created (#%d) but failed to get node ID: %s\n", created.Number, err)
				} else {
					query := fmt.Sprintf(`mutation { addSubIssue(input: {issueId: %q, subIssueId: %q}) { issue { id } } }`, parentID, childID)
					if _, err := c.GraphQL(context.Background(), query, nil); err != nil {
						fmt.Fprintf(os.Stderr, "warning: issue created (#%d) but failed to link as sub-issue: %s\n", created.Number, err)
					} else if !quietFlag {
						fmt.Fprintf(os.Stderr, "linked as sub-issue of #%d\n", issueParent)
					}
				}
			}
		}

		if !quietFlag && created.Number > 0 {
			fmt.Fprintf(os.Stderr, "created: #%d (%s)\n", created.Number, created.HTMLURL)
		}

		if flat := flattenIssue(data); flat != nil {
			out, _ := json.Marshal(flat)
			return printData("", out)
		}
		return printData("", data)
	},
}

var issuesEditCmd = &cobra.Command{
	Use:   "edit <number>",
	Short: "Edit an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		fields := map[string]any{}
		if issueTitle != "" {
			fields["title"] = issueTitle
		}

		if len(fields) > 0 {
			if _, err := c.Patch(context.Background(), "issues/"+args[0], fields); err != nil {
				return err
			}
		}

		// Labels: add and remove
		num, _ := parseNumber(args[0])
		for _, l := range issueAddLabel {
			c.Post(context.Background(), fmt.Sprintf("issues/%d/labels", num), map[string]any{"labels": []string{l}})
		}
		for _, l := range issueRemLabel {
			c.Delete(context.Background(), fmt.Sprintf("issues/%d/labels/%s", num, url.PathEscape(l)))
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "updated: #%s\n", args[0])
		}
		return nil
	},
}

var issuesCloseCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		body := map[string]any{"state": "closed"}
		if issueCloseReason != "" {
			body["state_reason"] = issueCloseReason
		}

		if _, err := c.Patch(context.Background(), "issues/"+args[0], body); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "closed: #%s\n", args[0])
		}
		return nil
	},
}

var issuesReopenCmd = &cobra.Command{
	Use:   "reopen <number>",
	Short: "Reopen a closed issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if _, err := c.Patch(context.Background(), "issues/"+args[0], map[string]any{"state": "open"}); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "reopened: #%s\n", args[0])
		}
		return nil
	},
}

var issuesAssignCmd = &cobra.Command{
	Use:   "assign <number>",
	Short: "Assign an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{"assignees": []string{issueUser}}
		if _, err := c.Post(context.Background(), "issues/"+args[0]+"/assignees", body); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "assigned: #%s → %s\n", args[0], issueUser)
		}
		return nil
	},
}

// resolveMilestoneNumber finds a milestone number by title.
func resolveMilestoneNumber(c *client.Client, title string) (int, error) {
	params := url.Values{"state": {"open"}, "per_page": {"100"}}
	data, err := c.Get(context.Background(), "milestones", params)
	if err != nil {
		return 0, err
	}
	var milestones []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	}
	if json.Unmarshal(data, &milestones) != nil {
		return 0, fmt.Errorf("milestone %q not found", title)
	}
	for _, m := range milestones {
		if strings.EqualFold(m.Title, title) {
			return m.Number, nil
		}
	}
	return 0, fmt.Errorf("milestone %q not found", title)
}

var issuesTimelineCmd = &cobra.Command{
	Use:   "timeline <number>",
	Short: "Show issue event history (like jx changelog)",
	Long: `Display timeline events: labels, assignments, status changes, PR links,
sub-issue events, milestones, renames.

Examples:
  gx issues timeline 123
  gx issues timeline 123 --limit 20`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		num, err := parseNumber(args[0])
		if err != nil {
			return err
		}

		query := fmt.Sprintf(`{
			repository(owner: %q, name: %q) {
				issue(number: %d) {
					timelineItems(first: %d, itemTypes: [
						LABELED_EVENT, UNLABELED_EVENT,
						ASSIGNED_EVENT, UNASSIGNED_EVENT,
						CLOSED_EVENT, REOPENED_EVENT,
						RENAMED_TITLE_EVENT,
						MILESTONED_EVENT, DEMILESTONED_EVENT,
						CROSS_REFERENCED_EVENT,
						SUB_ISSUE_ADDED_EVENT, SUB_ISSUE_REMOVED_EVENT,
						PARENT_ISSUE_ADDED_EVENT, PARENT_ISSUE_REMOVED_EVENT,
						BLOCKED_BY_ADDED_EVENT, BLOCKING_ADDED_EVENT
					]) {
						nodes {
							__typename
							... on LabeledEvent { label { name } actor { login } createdAt }
							... on UnlabeledEvent { label { name } actor { login } createdAt }
							... on AssignedEvent { assignee { ... on User { login } } actor { login } createdAt }
							... on UnassignedEvent { assignee { ... on User { login } } actor { login } createdAt }
							... on ClosedEvent { actor { login } stateReason createdAt }
							... on ReopenedEvent { actor { login } createdAt }
							... on RenamedTitleEvent { previousTitle currentTitle actor { login } createdAt }
							... on MilestonedEvent { milestoneTitle actor { login } createdAt }
							... on DemilestonedEvent { milestoneTitle actor { login } createdAt }
							... on CrossReferencedEvent {
								actor { login }
								createdAt
								source {
									... on PullRequest { number title state url }
									... on Issue { number title state url }
								}
							}
						}
					}
				}
			}
		}`, c.Owner(), c.Repo(), num, limitFlag)

		data, err := c.GraphQL(context.Background(), query, nil)
		if err != nil {
			return err
		}

		var resp struct {
			Repository struct {
				Issue struct {
					TimelineItems struct {
						Nodes []json.RawMessage `json:"nodes"`
					} `json:"timelineItems"`
				} `json:"issue"`
			} `json:"repository"`
		}
		json.Unmarshal(data, &resp)

		var rows []map[string]any
		for _, raw := range resp.Repository.Issue.TimelineItems.Nodes {
			var event struct {
				TypeName      string `json:"__typename"`
				Actor         *struct{ Login string } `json:"actor"`
				CreatedAt     string `json:"createdAt"`
				Label         *struct{ Name string } `json:"label"`
				Assignee      *struct{ Login string } `json:"assignee"`
				StateReason   string `json:"stateReason"`
				PreviousTitle string `json:"previousTitle"`
				CurrentTitle  string `json:"currentTitle"`
				MilestoneTitle string `json:"milestoneTitle"`
				Source        *struct {
					Number int    `json:"number"`
					Title  string `json:"title"`
					State  string `json:"state"`
					URL    string `json:"url"`
				} `json:"source"`
			}
			json.Unmarshal(raw, &event)

			row := map[string]any{
				"event":      event.TypeName,
				"created_at": event.CreatedAt,
			}
			if event.Actor != nil {
				row["actor"] = event.Actor.Login
			}
			if event.Label != nil {
				row["detail"] = event.Label.Name
			}
			if event.Assignee != nil {
				row["detail"] = event.Assignee.Login
			}
			if event.PreviousTitle != "" {
				row["detail"] = event.PreviousTitle + " → " + event.CurrentTitle
			}
			if event.MilestoneTitle != "" {
				row["detail"] = event.MilestoneTitle
			}
			if event.StateReason != "" {
				row["detail"] = event.StateReason
			}
			if event.Source != nil {
				row["detail"] = fmt.Sprintf("#%d %s (%s)", event.Source.Number, event.Source.Title, event.Source.State)
			}
			rows = append(rows, row)
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		fmt.Fprintf(os.Stderr, "timeline: %d events\n", len(rows))
		out, _ := json.Marshal(rows)
		return printData("", out)
	},
}

var issuesLinkedPRsCmd = &cobra.Command{
	Use:   "linked-prs <number>",
	Short: "Show pull requests linked to an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		num, err := parseNumber(args[0])
		if err != nil {
			return err
		}

		query := fmt.Sprintf(`{
			repository(owner: %q, name: %q) {
				issue(number: %d) {
					closedByPullRequestsReferences(first: 20) {
						nodes { number title state url merged }
					}
					timelineItems(first: 50, itemTypes: [CROSS_REFERENCED_EVENT]) {
						nodes {
							... on CrossReferencedEvent {
								source {
									... on PullRequest { number title state url merged }
								}
							}
						}
					}
				}
			}
		}`, c.Owner(), c.Repo(), num)

		data, err := c.GraphQL(context.Background(), query, nil)
		if err != nil {
			return err
		}

		var resp struct {
			Repository struct {
				Issue struct {
					ClosedByPRs struct {
						Nodes []struct {
							Number int    `json:"number"`
							Title  string `json:"title"`
							State  string `json:"state"`
							URL    string `json:"url"`
							Merged bool   `json:"merged"`
						} `json:"nodes"`
					} `json:"closedByPullRequestsReferences"`
					TimelineItems struct {
						Nodes []struct {
							Source *struct {
								Number int    `json:"number"`
								Title  string `json:"title"`
								State  string `json:"state"`
								URL    string `json:"url"`
								Merged bool   `json:"merged"`
							} `json:"source"`
						} `json:"nodes"`
					} `json:"timelineItems"`
				} `json:"issue"`
			} `json:"repository"`
		}
		json.Unmarshal(data, &resp)

		seen := map[int]bool{}
		var rows []map[string]any

		// Closing PRs first
		for _, pr := range resp.Repository.Issue.ClosedByPRs.Nodes {
			if pr.Number == 0 || seen[pr.Number] {
				continue
			}
			seen[pr.Number] = true
			state := strings.ToLower(pr.State)
			if pr.Merged {
				state = "merged"
			}
			rows = append(rows, map[string]any{
				"number":   pr.Number,
				"title":    pr.Title,
				"state":    state,
				"url":      pr.URL,
				"relation": "closes",
			})
		}

		// Cross-referenced PRs
		for _, item := range resp.Repository.Issue.TimelineItems.Nodes {
			if item.Source == nil || item.Source.Number == 0 || seen[item.Source.Number] {
				continue
			}
			seen[item.Source.Number] = true
			state := strings.ToLower(item.Source.State)
			if item.Source.Merged {
				state = "merged"
			}
			rows = append(rows, map[string]any{
				"number":   item.Source.Number,
				"title":    item.Source.Title,
				"state":    state,
				"url":      item.Source.URL,
				"relation": "references",
			})
		}

		if rows == nil {
			rows = []map[string]any{}
		}
		fmt.Fprintf(os.Stderr, "linked PRs: %d\n", len(rows))
		out, _ := json.Marshal(rows)
		return printData("", out)
	},
}

var issuesPinCmd = &cobra.Command{
	Use:   "pin <number>",
	Short: "Pin an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		num, _ := parseNumber(args[0])
		id, err := c.IssueNodeID(context.Background(), num)
		if err != nil {
			return err
		}
		query := fmt.Sprintf(`mutation { pinIssue(input: {issueId: %q}) { issue { title } } }`, id)
		if _, err := c.GraphQL(context.Background(), query, nil); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "pinned: #%d\n", num)
		}
		return nil
	},
}

var issuesUnpinCmd = &cobra.Command{
	Use:   "unpin <number>",
	Short: "Unpin an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		num, _ := parseNumber(args[0])
		id, err := c.IssueNodeID(context.Background(), num)
		if err != nil {
			return err
		}
		query := fmt.Sprintf(`mutation { unpinIssue(input: {issueId: %q}) { issue { title } } }`, id)
		if _, err := c.GraphQL(context.Background(), query, nil); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "unpinned: #%d\n", num)
		}
		return nil
	},
}

var issuesLockCmd = &cobra.Command{
	Use:   "lock <number>",
	Short: "Lock issue conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		// Lock uses PUT
		if _, err := c.Put(context.Background(), "issues/"+args[0]+"/lock", map[string]any{"lock_reason": "resolved"}); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "locked: #%s\n", args[0])
		}
		return nil
	},
}

var issuesUnlockCmd = &cobra.Command{
	Use:   "unlock <number>",
	Short: "Unlock issue conversation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "issues/"+args[0]+"/lock"); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "unlocked: #%s\n", args[0])
		}
		return nil
	},
}
