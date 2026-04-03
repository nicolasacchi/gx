package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	milestoneTitle string
	milestoneDue   string
	milestoneDesc  string
)

func init() {
	rootCmd.AddCommand(milestonesCmd)
	milestonesCmd.AddCommand(milestonesListCmd)
	milestonesCmd.AddCommand(milestonesGetCmd)
	milestonesCmd.AddCommand(milestonesCreateCmd)
	milestonesCmd.AddCommand(milestonesEditCmd)
	milestonesCmd.AddCommand(milestonesCloseCmd)
	milestonesCmd.AddCommand(milestonesReopenCmd)
	milestonesCmd.AddCommand(milestonesIssuesCmd)

	milestonesCreateCmd.Flags().StringVar(&milestoneTitle, "title", "", "Milestone title (required)")
	milestonesCreateCmd.Flags().StringVar(&milestoneDue, "due", "", "Due date (YYYY-MM-DD)")
	milestonesCreateCmd.Flags().StringVar(&milestoneDesc, "description", "", "Description")
	milestonesCreateCmd.MarkFlagRequired("title")

	milestonesEditCmd.Flags().StringVar(&milestoneTitle, "title", "", "New title")
	milestonesEditCmd.Flags().StringVar(&milestoneDue, "due", "", "New due date")
	milestonesEditCmd.Flags().StringVar(&milestoneDesc, "description", "", "New description")
}

var milestonesCmd = &cobra.Command{
	Use:     "milestones",
	Aliases: []string{"ms"},
	Short:   "Manage milestones (epic equivalent)",
}

var milestonesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List milestones",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{
			"state":     {"all"},
			"per_page":  {strconv.Itoa(limitFlag)},
			"sort":      {"due_on"},
			"direction": {"asc"},
		}
		data, err := c.Get(context.Background(), "milestones", params)
		if err != nil {
			return err
		}

		var milestones []json.RawMessage
		json.Unmarshal(data, &milestones)
		fmt.Fprintf(os.Stderr, "milestones: %d\n", len(milestones))

		var rows []map[string]any
		for _, raw := range milestones {
			var m struct {
				Number       int    `json:"number"`
				Title        string `json:"title"`
				State        string `json:"state"`
				Description  string `json:"description"`
				DueOn        string `json:"due_on"`
				OpenIssues   int    `json:"open_issues"`
				ClosedIssues int    `json:"closed_issues"`
			}
			json.Unmarshal(raw, &m)
			rows = append(rows, map[string]any{
				"number":        m.Number,
				"title":         m.Title,
				"state":         m.State,
				"description":   m.Description,
				"due_on":        m.DueOn,
				"open_issues":   m.OpenIssues,
				"closed_issues": m.ClosedIssues,
			})
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		out, _ := json.Marshal(rows)
		return printData("milestones.list", out)
	},
}

var milestonesGetCmd = &cobra.Command{
	Use:   "get <number>",
	Short: "Get milestone details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		data, err := c.Get(context.Background(), "milestones/"+args[0], nil)
		if err != nil {
			return err
		}
		return printData("", data)
	},
}

var milestonesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a milestone",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{"title": milestoneTitle}
		if milestoneDue != "" {
			body["due_on"] = milestoneDue + "T00:00:00Z"
		}
		if milestoneDesc != "" {
			body["description"] = milestoneDesc
		}
		data, err := c.Post(context.Background(), "milestones", body)
		if err != nil {
			return err
		}
		var created struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
		}
		json.Unmarshal(data, &created)
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "created milestone: #%d %s\n", created.Number, created.Title)
		}
		return printData("", data)
	},
}

var milestonesEditCmd = &cobra.Command{
	Use:   "edit <number>",
	Short: "Edit a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{}
		if milestoneTitle != "" {
			body["title"] = milestoneTitle
		}
		if milestoneDue != "" {
			body["due_on"] = milestoneDue + "T00:00:00Z"
		}
		if milestoneDesc != "" {
			body["description"] = milestoneDesc
		}
		if _, err := c.Patch(context.Background(), "milestones/"+args[0], body); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "updated milestone: #%s\n", args[0])
		}
		return nil
	},
}

var milestonesCloseCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if _, err := c.Patch(context.Background(), "milestones/"+args[0], map[string]any{"state": "closed"}); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "closed milestone: #%s\n", args[0])
		}
		return nil
	},
}

var milestonesReopenCmd = &cobra.Command{
	Use:   "reopen <number>",
	Short: "Reopen a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if _, err := c.Patch(context.Background(), "milestones/"+args[0], map[string]any{"state": "open"}); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "reopened milestone: #%s\n", args[0])
		}
		return nil
	},
}

var milestonesIssuesCmd = &cobra.Command{
	Use:   "issues <number>",
	Short: "List issues in a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{
			"milestone": {args[0]},
			"state":     {"all"},
			"per_page":  {strconv.Itoa(limitFlag)},
		}
		data, err := c.Get(context.Background(), "issues", params)
		if err != nil {
			return err
		}
		return printData("issues.list", flattenIssues(data))
	},
}
