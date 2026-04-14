package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	subIssueTitle string
	subIssueLabel []string
	subIssueAfter int
)

func init() {
	rootCmd.AddCommand(subIssuesCmd)
	subIssuesCmd.AddCommand(subIssuesListCmd)
	subIssuesCmd.AddCommand(subIssuesAddCmd)
	subIssuesCmd.AddCommand(subIssuesRemoveCmd)
	subIssuesCmd.AddCommand(subIssuesReorderCmd)

	subIssuesAddCmd.Flags().StringVar(&subIssueTitle, "title", "", "Create a new issue and link as sub-issue")
	subIssuesAddCmd.Flags().StringSliceVar(&subIssueLabel, "label", nil, "Labels for the new sub-issue (requires --title)")

	subIssuesReorderCmd.Flags().IntVar(&subIssueAfter, "after", 0, "Place after this sibling issue number")
}

var subIssuesCmd = &cobra.Command{
	Use:     "sub-issues",
	Aliases: []string{"si"},
	Short:   "Manage sub-issues (parent-child relationships)",
}

var subIssuesListCmd = &cobra.Command{
	Use:   "list <parent-number>",
	Short: "List sub-issues of a parent",
	Long: `List all sub-issues of a parent issue.

Examples:
  gx sub-issues list 123
  gx si list 123 --jq '#.{number:number,title:title,state:state}'`,
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
					title
					subIssues(first: 100) {
						totalCount
						nodes {
							number
							title
							state
							url
							assignees(first: 5) { nodes { login } }
							labels(first: 10) { nodes { name } }
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
					Title     string `json:"title"`
					SubIssues struct {
						TotalCount int `json:"totalCount"`
						Nodes      []struct {
							Number    int    `json:"number"`
							Title     string `json:"title"`
							State     string `json:"state"`
							URL       string `json:"url"`
							Assignees struct {
								Nodes []struct{ Login string } `json:"nodes"`
							} `json:"assignees"`
							Labels struct {
								Nodes []struct{ Name string } `json:"nodes"`
							} `json:"labels"`
						} `json:"nodes"`
					} `json:"subIssues"`
				} `json:"issue"`
			} `json:"repository"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "sub-issues of #%d (%s): %d\n", num, resp.Repository.Issue.Title, resp.Repository.Issue.SubIssues.TotalCount)

		var rows []map[string]any
		for _, si := range resp.Repository.Issue.SubIssues.Nodes {
			row := map[string]any{
				"number": si.Number,
				"title":  si.Title,
				"state":  strings.ToLower(si.State),
				"url":    si.URL,
			}
			if len(si.Assignees.Nodes) > 0 {
				row["assignee"] = si.Assignees.Nodes[0].Login
			}
			if len(si.Labels.Nodes) > 0 {
				labels := make([]string, len(si.Labels.Nodes))
				for i, l := range si.Labels.Nodes {
					labels[i] = l.Name
				}
				row["labels"] = labels
			}
			rows = append(rows, row)
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		out, _ := json.Marshal(rows)
		return printData("sub-issues.list", out)
	},
}

var subIssuesAddCmd = &cobra.Command{
	Use:   "add <parent-number> [child-number]",
	Short: "Add a sub-issue to a parent",
	Long: `Link an existing issue as a sub-issue, or create a new one.

Examples:
  gx sub-issues add 123 456                                    # link existing #456 as child of #123
  gx sub-issues add 123 --title "Brand model" --label "type:sub-task"  # create + link`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		parentNum, err := parseNumber(args[0])
		if err != nil {
			return err
		}

		var childNum int

		if len(args) == 2 {
			// Link existing issue
			childNum, err = parseNumber(args[1])
			if err != nil {
				return err
			}
		} else if subIssueTitle != "" {
			// Create new issue first
			fields := map[string]any{"title": subIssueTitle}
			if len(subIssueLabel) > 0 {
				fields["labels"] = subIssueLabel
			}
			data, err := c.Post(context.Background(), "issues", fields)
			if err != nil {
				return fmt.Errorf("create sub-issue: %w", err)
			}
			var created struct{ Number int }
			json.Unmarshal(data, &created)
			childNum = created.Number
			if !quietFlag {
				fmt.Fprintf(os.Stderr, "created: #%d\n", childNum)
			}
		} else {
			return fmt.Errorf("provide child issue number or --title to create a new one")
		}

		// Resolve node IDs
		parentID, err := c.IssueNodeID(context.Background(), parentNum)
		if err != nil {
			return fmt.Errorf("resolve parent #%d: %w", parentNum, err)
		}
		childID, err := c.IssueNodeID(context.Background(), childNum)
		if err != nil {
			return fmt.Errorf("resolve child #%d: %w", childNum, err)
		}

		// Link as sub-issue
		query := fmt.Sprintf(`mutation {
			addSubIssue(input: {issueId: %q, subIssueId: %q}) {
				issue { title }
				subIssue { number title }
			}
		}`, parentID, childID)

		data, err := c.GraphQL(context.Background(), query, nil)
		if err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "linked: #%d → sub-issue of #%d\n", childNum, parentNum)
		}
		// Only emit the mutation JSON to stdout if the caller explicitly asked for it
		// via --json or --jq. See itemsAddCmd for rationale.
		if jsonFlag || jqFlag != "" {
			return printData("", data)
		}
		return nil
	},
}

var subIssuesRemoveCmd = &cobra.Command{
	Use:   "remove <parent-number> <child-number>",
	Short: "Remove a sub-issue from a parent",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		parentNum, _ := parseNumber(args[0])
		childNum, _ := parseNumber(args[1])

		parentID, err := c.IssueNodeID(context.Background(), parentNum)
		if err != nil {
			return err
		}
		childID, err := c.IssueNodeID(context.Background(), childNum)
		if err != nil {
			return err
		}

		query := fmt.Sprintf(`mutation {
			removeSubIssue(input: {issueId: %q, subIssueId: %q}) {
				issue { title }
			}
		}`, parentID, childID)

		if _, err := c.GraphQL(context.Background(), query, nil); err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "unlinked: #%d from parent #%d\n", childNum, parentNum)
		}
		return nil
	},
}

var subIssuesReorderCmd = &cobra.Command{
	Use:   "reorder <parent-number> <child-number>",
	Short: "Reorder a sub-issue within its parent",
	Long: `Move a sub-issue to a different position.

Examples:
  gx sub-issues reorder 123 456 --after 789`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		parentNum, _ := parseNumber(args[0])
		childNum, _ := parseNumber(args[1])

		parentID, err := c.IssueNodeID(context.Background(), parentNum)
		if err != nil {
			return err
		}
		childID, err := c.IssueNodeID(context.Background(), childNum)
		if err != nil {
			return err
		}

		afterClause := ""
		if subIssueAfter > 0 {
			afterID, err := c.IssueNodeID(context.Background(), subIssueAfter)
			if err != nil {
				return err
			}
			afterClause = fmt.Sprintf(`, afterId: %q`, afterID)
		}

		query := fmt.Sprintf(`mutation {
			reprioritizeSubIssue(input: {issueId: %q, subIssueId: %q%s}) {
				issue { title }
			}
		}`, parentID, childID, afterClause)

		if _, err := c.GraphQL(context.Background(), query, nil); err != nil {
			return err
		}

		if !quietFlag {
			fmt.Fprintf(os.Stderr, "reordered: #%d in parent #%d\n", childNum, parentNum)
		}
		return nil
	},
}
