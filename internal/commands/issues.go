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
	issueState           string
	issueLabel           []string
	issueMilestone       string
	issueRemoveMilestone bool
	issueAssignee        string
	issueAssignees       []string
	issueAddAssignee     []string
	issueRemAssignee     []string
	issueTitle           string
	issueBody            string
	issueBodyFile        string
	issueType            string
	issueEditState       string
	issueParent          int
	issueAddLabel        []string
	issueRemLabel        []string
	issueCloseReason     string
	issueReopenReason    string
	issueTransferTo      string
	issueDevelopName     string
	issueUser            string

	// create --project-number board flags (custom fields reuse itemsFields/itemsValues)
	createProjectNum int
	createStatus     string
	createPriority   string
	createPoints     float64
	createIteration  string
)

func init() {
	rootCmd.AddCommand(issuesCmd)
	issuesCmd.AddCommand(issuesListCmd)
	issuesCmd.AddCommand(issuesGetCmd)
	issuesCmd.AddCommand(issuesCreateCmd)
	issuesCmd.AddCommand(issuesTypesCmd)
	issuesCmd.AddCommand(issuesTransferCmd)
	issuesCmd.AddCommand(issuesDevelopCmd)
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
	issuesCreateCmd.Flags().StringSliceVar(&issueAssignees, "assignee", nil, "Assignee login(s); repeatable or comma-separated")
	issuesCreateCmd.Flags().StringVar(&issueType, "type", "", "Issue type name (e.g. Task, Bug, Feature)")
	issuesCreateCmd.Flags().IntVar(&issueParent, "parent", 0, "Parent issue number (creates as sub-issue)")
	issuesCreateCmd.Flags().IntVar(&createProjectNum, "project-number", 0, "Add the new issue to this project board")
	issuesCreateCmd.Flags().StringVar(&createStatus, "status", "", "Board Status (needs --project-number)")
	issuesCreateCmd.Flags().StringVar(&createPriority, "priority", "", "Board Priority (needs --project-number)")
	issuesCreateCmd.Flags().Float64Var(&createPoints, "points", 0, "Board Story Points (needs --project-number)")
	issuesCreateCmd.Flags().StringVar(&createIteration, "iteration", "", "Board iteration/Sprint by title (needs --project-number)")
	issuesCreateCmd.Flags().StringArrayVar(&itemsFields, "field", nil, "Board custom field name (repeatable; pair with --value; needs --project-number)")
	issuesCreateCmd.Flags().StringArrayVar(&itemsValues, "value", nil, "Value for the matching --field (repeatable)")
	issuesCreateCmd.MarkFlagRequired("title")

	issuesEditCmd.Flags().StringVar(&issueTitle, "title", "", "New title")
	issuesEditCmd.Flags().StringVar(&issueBody, "body", "", "New body")
	issuesEditCmd.Flags().StringVar(&issueBodyFile, "body-file", "", "Read new body from file")
	issuesEditCmd.Flags().StringVar(&issueType, "type", "", "Set issue type name (e.g. Task, Bug, Feature)")
	issuesEditCmd.Flags().StringVar(&issueMilestone, "milestone", "", "Set milestone by title")
	issuesEditCmd.Flags().BoolVar(&issueRemoveMilestone, "remove-milestone", false, "Clear the milestone")
	issuesEditCmd.Flags().StringVar(&issueEditState, "state", "", "Set state: open or closed")
	issuesEditCmd.Flags().StringSliceVar(&issueAssignees, "assignee", nil, "Replace assignees with this set; repeatable or comma-separated")
	issuesEditCmd.Flags().StringSliceVar(&issueAddAssignee, "add-assignee", nil, "Add assignees; repeatable")
	issuesEditCmd.Flags().StringSliceVar(&issueRemAssignee, "remove-assignee", nil, "Remove assignees; repeatable")
	issuesEditCmd.Flags().StringSliceVar(&issueAddLabel, "add-label", nil, "Add labels")
	issuesEditCmd.Flags().StringSliceVar(&issueRemLabel, "remove-label", nil, "Remove labels")

	issuesCloseCmd.Flags().StringVar(&issueCloseReason, "reason", "", "Close reason: completed, not_planned")

	issuesReopenCmd.Flags().StringVar(&issueReopenReason, "reason", "reopened", "Reopen state_reason (default: reopened)")

	issuesTransferCmd.Flags().StringVar(&issueTransferTo, "to-repo", "", "Target repository name, same owner (required)")
	issuesTransferCmd.MarkFlagRequired("to-repo")

	issuesDevelopCmd.Flags().StringVar(&issueDevelopName, "name", "", "Branch name (default: <number>-<slugified title>)")

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
			"state":     {issueState},
			"per_page":  {strconv.Itoa(limitFlag)},
			"sort":      {"updated"},
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
  gx issues create --title "Fix login bug" --type "Bug" --label "type:bug"
  gx issues create --title "Phase 1" --type "Task" --milestone "CoMarketing" --parent 456`,
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
		if len(issueAssignees) > 0 {
			fields["assignees"] = issueAssignees
		}
		if issueType != "" {
			fields["type"] = issueType
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

		// If --project-number specified, add to the board and apply any board fields.
		if createProjectNum > 0 && created.Number > 0 {
			bf := boardFields{
				Status:    createStatus,
				Priority:  createPriority,
				Points:    createPoints,
				Iteration: createIteration,
				Fields:    itemsFields,
				Values:    itemsValues,
			}
			if err := addToBoardAndSet(c, createProjectNum, created.Number, bf); err != nil {
				fmt.Fprintf(os.Stderr, "warning: issue created (#%d) but board update failed: %s\n", created.Number, err)
			} else if !quietFlag {
				fmt.Fprintf(os.Stderr, "added #%d to project %d\n", created.Number, createProjectNum)
			}
		}

		warnAssigneeCap(issueAssignees)
		warnDroppedAssignees(data, issueAssignees)

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
	Long: `Edit an existing issue's fields. Any combination of flags may be set in one call.

Examples:
  gx issues edit 123 --title "New title" --type "Bug"
  gx issues edit 123 --body-file notes.md --milestone "v2.1"
  gx issues edit 123 --assignee alice --assignee bob       # replace assignee set
  gx issues edit 123 --add-assignee carol --remove-assignee bob
  gx issues edit 123 --state closed --add-label "done"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		fields := map[string]any{}
		if issueTitle != "" {
			fields["title"] = issueTitle
		}
		body := issueBody
		if issueBodyFile != "" {
			content, err := os.ReadFile(issueBodyFile)
			if err != nil {
				return fmt.Errorf("read body file: %w", err)
			}
			body = string(content)
		}
		if body != "" {
			fields["body"] = body
		}
		if issueType != "" {
			fields["type"] = issueType
		}
		if issueEditState != "" {
			if issueEditState != "open" && issueEditState != "closed" {
				return fmt.Errorf("--state must be 'open' or 'closed', got %q", issueEditState)
			}
			fields["state"] = issueEditState
			if issueEditState == "open" {
				fields["state_reason"] = "reopened"
			}
		}
		if issueRemoveMilestone {
			fields["milestone"] = nil
		} else if issueMilestone != "" {
			milestoneNum, err := resolveMilestoneNumber(c, issueMilestone)
			if err != nil {
				return err
			}
			fields["milestone"] = milestoneNum
		}
		// --assignee replaces the whole assignee set (PATCH semantics).
		if len(issueAssignees) > 0 {
			fields["assignees"] = issueAssignees
		}

		if len(fields) > 0 {
			if _, err := c.Patch(context.Background(), "issues/"+args[0], fields); err != nil {
				return err
			}
		}

		num, _ := parseNumber(args[0])
		// Assignees: incremental add/remove via the dedicated sub-endpoints.
		if len(issueAddAssignee) > 0 {
			if _, err := c.Post(context.Background(), fmt.Sprintf("issues/%d/assignees", num), map[string]any{"assignees": issueAddAssignee}); err != nil {
				return fmt.Errorf("add assignees: %w", err)
			}
		}
		if len(issueRemAssignee) > 0 {
			if _, err := c.DeleteBody(context.Background(), fmt.Sprintf("issues/%d/assignees", num), map[string]any{"assignees": issueRemAssignee}); err != nil {
				return fmt.Errorf("remove assignees: %w", err)
			}
		}
		// Labels: add and remove (surface errors — don't claim success on a failed call).
		for _, l := range issueAddLabel {
			if _, err := c.Post(context.Background(), fmt.Sprintf("issues/%d/labels", num), map[string]any{"labels": []string{l}}); err != nil {
				return fmt.Errorf("add label %q: %w", l, err)
			}
		}
		for _, l := range issueRemLabel {
			if err := c.Delete(context.Background(), fmt.Sprintf("issues/%d/labels/%s", num, url.PathEscape(l))); err != nil {
				return fmt.Errorf("remove label %q: %w", l, err)
			}
		}

		// Re-fetch so we can echo the updated issue and verify assignees landed.
		data, err := c.Get(context.Background(), "issues/"+args[0], nil)
		if err != nil {
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "updated: #%s\n", args[0])
			}
			return nil
		}
		warnAssigneeCap(issueAssignees)
		warnDroppedAssignees(data, append(append([]string{}, issueAssignees...), issueAddAssignee...))
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "updated: #%s\n", args[0])
		}
		if flat := flattenIssue(data); flat != nil {
			out, _ := json.Marshal(flat)
			return printData("", out)
		}
		return printData("", data)
	},
}

// warnAssigneeCap notes when more than GitHub's hard cap of 10 assignees is
// requested, since the extras are silently dropped.
func warnAssigneeCap(logins []string) {
	if len(logins) > 10 {
		fmt.Fprintf(os.Stderr, "warning: %d assignees requested but GitHub caps issues at 10 — extras will be dropped\n", len(logins))
	}
}

// warnDroppedAssignees warns to stderr for any requested assignee login that is
// absent from the issue's final assignee set — GitHub silently ignores logins
// that aren't valid assignees for the repo (no error), so this surfaces the loss.
func warnDroppedAssignees(data json.RawMessage, requested []string) {
	if len(requested) == 0 {
		return
	}
	var issue struct {
		Assignees []struct{ Login string } `json:"assignees"`
	}
	if json.Unmarshal(data, &issue) != nil {
		return
	}
	have := make(map[string]bool, len(issue.Assignees))
	for _, a := range issue.Assignees {
		have[strings.ToLower(a.Login)] = true
	}
	for _, r := range requested {
		if !have[strings.ToLower(r)] {
			fmt.Fprintf(os.Stderr, "warning: assignee %q was not applied (not a valid assignee for this repo?)\n", r)
		}
	}
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
		body := map[string]any{"state": "open"}
		if issueReopenReason != "" {
			body["state_reason"] = issueReopenReason
		}
		if _, err := c.Patch(context.Background(), "issues/"+args[0], body); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "reopened: #%s\n", args[0])
		}
		return nil
	},
}

var issuesAssignCmd = &cobra.Command{
	Use:        "assign <number>",
	Short:      "Assign an issue",
	Deprecated: "use `gx issues edit <n> --add-assignee <login>` (repeatable) instead.",
	Args:       cobra.ExactArgs(1),
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

// resolveMilestoneNumber finds a milestone number by title across all states,
// paginating beyond the 100/page cap (so closed and 101st+ milestones resolve).
func resolveMilestoneNumber(c *client.Client, title string) (int, error) {
	for page := 1; ; page++ {
		params := url.Values{"state": {"all"}, "per_page": {"100"}, "page": {strconv.Itoa(page)}}
		data, err := c.Get(context.Background(), "milestones", params)
		if err != nil {
			return 0, err
		}
		var milestones []struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
		}
		if json.Unmarshal(data, &milestones) != nil || len(milestones) == 0 {
			break
		}
		for _, m := range milestones {
			if strings.EqualFold(m.Title, title) {
				return m.Number, nil
			}
		}
		if len(milestones) < 100 {
			break
		}
	}
	return 0, fmt.Errorf("milestone %q not found", title)
}

var issuesTransferCmd = &cobra.Command{
	Use:   "transfer <number>",
	Short: "Transfer an issue to another repository (same owner)",
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
		issueID, err := c.IssueNodeID(context.Background(), num)
		if err != nil {
			return err
		}
		repoQuery := fmt.Sprintf(`{ repository(owner: %q, name: %q) { id } }`, c.Owner(), issueTransferTo)
		repoData, err := c.GraphQL(context.Background(), repoQuery, nil)
		if err != nil {
			return err
		}
		var rr struct {
			Repository struct {
				ID string `json:"id"`
			} `json:"repository"`
		}
		if json.Unmarshal(repoData, &rr) != nil || rr.Repository.ID == "" {
			return fmt.Errorf("target repository %q not found under %s", issueTransferTo, c.Owner())
		}
		mutation := fmt.Sprintf(`mutation { transferIssue(input: {issueId: %q, repositoryId: %q}) { issue { number url } } }`, issueID, rr.Repository.ID)
		data, err := c.GraphQL(context.Background(), mutation, nil)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "transferred #%d → %s/%s\n", num, c.Owner(), issueTransferTo)
		}
		return printData("", data)
	},
}

var issuesDevelopCmd = &cobra.Command{
	Use:   "develop <number>",
	Short: "Create a branch linked to an issue (off the default branch)",
	Long: `Create a Git branch linked to an issue (like 'gh issue develop'), branched
from the repo's default branch tip. Defaults the branch name to
<number>-<slugified title> unless --name is given.

Example:
  gx issues develop 123 --name 123-fix-login`,
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
		infoQuery := fmt.Sprintf(`{
			repository(owner: %q, name: %q) {
				defaultBranchRef { target { oid } }
				issue(number: %d) { id title }
			}
		}`, c.Owner(), c.Repo(), num)
		infoData, err := c.GraphQL(context.Background(), infoQuery, nil)
		if err != nil {
			return err
		}
		var info struct {
			Repository struct {
				DefaultBranchRef struct {
					Target struct {
						Oid string `json:"oid"`
					} `json:"target"`
				} `json:"defaultBranchRef"`
				Issue struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"issue"`
			} `json:"repository"`
		}
		if json.Unmarshal(infoData, &info) != nil || info.Repository.Issue.ID == "" || info.Repository.DefaultBranchRef.Target.Oid == "" {
			return fmt.Errorf("could not resolve issue #%d or the default branch in %s/%s", num, c.Owner(), c.Repo())
		}
		name := issueDevelopName
		if name == "" {
			name = fmt.Sprintf("%d-%s", num, slugify(info.Repository.Issue.Title))
		}
		mutation := fmt.Sprintf(`mutation {
			createLinkedBranch(input: {issueId: %q, oid: %q, name: %q}) {
				linkedBranch { ref { name } }
			}
		}`, info.Repository.Issue.ID, info.Repository.DefaultBranchRef.Target.Oid, name)
		data, err := c.GraphQL(context.Background(), mutation, nil)
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "created branch %q linked to #%d\n", name, num)
		}
		return printData("", data)
	},
}

// slugify turns an issue title into a branch-name-safe slug.
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 50 {
		out = strings.Trim(out[:50], "-")
	}
	return out
}

var issuesTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List the org's custom issue types (valid --type values)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.GetAbsolute(context.Background(), fmt.Sprintf("https://api.github.com/orgs/%s/issue-types", c.Owner()))
		if err != nil {
			return err
		}
		var types []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if json.Unmarshal(data, &types) == nil && len(types) > 0 {
			flat := make([]map[string]any, len(types))
			for i, t := range types {
				flat[i] = map[string]any{"name": t.Name, "description": t.Description}
			}
			out, _ := json.Marshal(flat)
			return printData("issues.types", out)
		}
		return printData("", data)
	},
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
				TypeName       string                  `json:"__typename"`
				Actor          *struct{ Login string } `json:"actor"`
				CreatedAt      string                  `json:"createdAt"`
				Label          *struct{ Name string }  `json:"label"`
				Assignee       *struct{ Login string } `json:"assignee"`
				StateReason    string                  `json:"stateReason"`
				PreviousTitle  string                  `json:"previousTitle"`
				CurrentTitle   string                  `json:"currentTitle"`
				MilestoneTitle string                  `json:"milestoneTitle"`
				Source         *struct {
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
