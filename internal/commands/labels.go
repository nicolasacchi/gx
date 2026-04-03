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
	labelName  string
	labelColor string
	labelDesc  string
)

func init() {
	rootCmd.AddCommand(labelsCmd)
	labelsCmd.AddCommand(labelsListCmd)
	labelsCmd.AddCommand(labelsCreateCmd)
	labelsCmd.AddCommand(labelsDeleteCmd)

	labelsCreateCmd.Flags().StringVar(&labelName, "name", "", "Label name (required)")
	labelsCreateCmd.Flags().StringVar(&labelColor, "color", "", "Label color hex (without #)")
	labelsCreateCmd.Flags().StringVar(&labelDesc, "description", "", "Label description")
	labelsCreateCmd.MarkFlagRequired("name")
}

var labelsCmd = &cobra.Command{
	Use:   "labels",
	Short: "Manage repository labels",
}

var labelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List labels",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		params := url.Values{"per_page": {strconv.Itoa(limitFlag)}}
		data, err := c.Get(context.Background(), "labels", params)
		if err != nil {
			return err
		}

		var labels []json.RawMessage
		json.Unmarshal(data, &labels)
		fmt.Fprintf(os.Stderr, "labels: %d\n", len(labels))

		var rows []map[string]any
		for _, raw := range labels {
			var l struct {
				Name        string `json:"name"`
				Color       string `json:"color"`
				Description string `json:"description"`
			}
			json.Unmarshal(raw, &l)
			rows = append(rows, map[string]any{
				"name":        l.Name,
				"color":       l.Color,
				"description": l.Description,
			})
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		out, _ := json.Marshal(rows)
		return printData("labels.list", out)
	},
}

var labelsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a label",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		body := map[string]any{"name": labelName}
		if labelColor != "" {
			body["color"] = labelColor
		}
		if labelDesc != "" {
			body["description"] = labelDesc
		}
		if _, err := c.Post(context.Background(), "labels", body); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "label created: %s\n", labelName)
		}
		return nil
	},
}

var labelsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a label",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		if err := c.Delete(context.Background(), "labels/"+url.PathEscape(args[0])); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(os.Stderr, "label deleted: %s\n", args[0])
		}
		return nil
	},
}
