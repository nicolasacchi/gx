package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	searchLabel     []string
	searchState     string
	searchMilestone string
)

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringSliceVar(&searchLabel, "label", nil, "Filter by labels")
	searchCmd.Flags().StringVar(&searchState, "state", "", "Filter by state: open, closed")
	searchCmd.Flags().StringVar(&searchMilestone, "milestone", "", "Filter by milestone")
}

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search issues",
	Long: `Search issues by text query and/or filters.

Examples:
  gx search "coupon discount"
  gx search --label "type:bug" --state open
  gx search --milestone "CoMarketing Migration"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		// Build GitHub search query
		q := fmt.Sprintf("repo:%s/%s is:issue", c.Owner(), c.Repo())
		if len(args) > 0 {
			q += " " + strings.Join(args, " ")
		}
		if searchState != "" {
			q += " state:" + searchState
		}
		for _, l := range searchLabel {
			q += fmt.Sprintf(` label:"%s"`, l)
		}
		if searchMilestone != "" {
			q += fmt.Sprintf(` milestone:"%s"`, searchMilestone)
		}

		// Use the search API (not repo issues API — supports full text)
		params := url.Values{
			"q":        {q},
			"per_page": {strconv.Itoa(limitFlag)},
			"sort":     {"updated"},
			"order":    {"desc"},
		}

		data, err := c.GetAbsolute(context.Background(), "https://api.github.com/search/issues?"+params.Encode())
		if err != nil {
			return err
		}

		var resp struct {
			TotalCount int               `json:"total_count"`
			Items      []json.RawMessage `json:"items"`
		}
		json.Unmarshal(data, &resp)
		fmt.Fprintf(os.Stderr, "search: %d total results\n", resp.TotalCount)

		return printData("search", flattenIssues(json.RawMessage(mustMarshal(resp.Items))))
	},
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
