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
	commentBody     string
	commentBodyFile string
)

func init() {
	rootCmd.AddCommand(commentsCmd)
	commentsCmd.AddCommand(commentsListCmd)
	commentsCmd.AddCommand(commentsAddCmd)

	commentsAddCmd.Flags().StringVar(&commentBody, "body", "", "Comment body")
	commentsAddCmd.Flags().StringVar(&commentBodyFile, "file", "", "Read body from file")
}

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Manage issue comments",
}

var commentsListCmd = &cobra.Command{
	Use:   "list <issue-number>",
	Short: "List comments on an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{"per_page": {strconv.Itoa(limitFlag)}}
		data, err := c.Get(context.Background(), "issues/"+args[0]+"/comments", params)
		if err != nil {
			return err
		}

		var comments []json.RawMessage
		json.Unmarshal(data, &comments)
		fmt.Fprintf(os.Stderr, "comments: %d\n", len(comments))

		var rows []map[string]any
		for _, raw := range comments {
			var cm struct {
				ID        int    `json:"id"`
				User      struct{ Login string } `json:"user"`
				Body      string `json:"body"`
				CreatedAt string `json:"created_at"`
			}
			json.Unmarshal(raw, &cm)
			body := cm.Body
			if len(body) > 200 {
				body = body[:197] + "..."
			}
			rows = append(rows, map[string]any{
				"id":         cm.ID,
				"user":       cm.User.Login,
				"body":       body,
				"created_at": cm.CreatedAt,
			})
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		out, _ := json.Marshal(rows)
		return printData("comments.list", out)
	},
}

var commentsAddCmd = &cobra.Command{
	Use:   "add <issue-number>",
	Short: "Add a comment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		body := commentBody
		if commentBodyFile != "" {
			content, err := os.ReadFile(commentBodyFile)
			if err != nil {
				return err
			}
			body = string(content)
		}
		if body == "" {
			return fmt.Errorf("provide --body or --file")
		}

		data, err := c.Post(context.Background(), "issues/"+args[0]+"/comments", map[string]any{"body": body})
		if err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "comment added on #%s\n", args[0])
		}
		return printData("", data)
	},
}
