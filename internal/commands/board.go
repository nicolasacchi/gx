package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var boardProjectNum int

func init() {
	rootCmd.AddCommand(boardCmd)
	boardCmd.AddCommand(boardListCmd)
	boardCmd.AddCommand(boardFieldsCmd)

	boardFieldsCmd.Flags().IntVar(&boardProjectNum, "project-number", 0, "Project number (required)")
	boardFieldsCmd.MarkFlagRequired("project-number")
}

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Manage project boards",
}

var boardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List org projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		query := fmt.Sprintf(`{
			organization(login: %q) {
				projectsV2(first: 20) {
					nodes {
						number
						title
						closed
						url
					}
				}
			}
		}`, c.Owner())

		data, err := c.GraphQL(context.Background(), query, nil)
		if err != nil {
			return err
		}

		var resp struct {
			Organization struct {
				ProjectsV2 struct {
					Nodes []struct {
						Number int    `json:"number"`
						Title  string `json:"title"`
						Closed bool   `json:"closed"`
						URL    string `json:"url"`
					} `json:"nodes"`
				} `json:"projectsV2"`
			} `json:"organization"`
		}
		json.Unmarshal(data, &resp)

		var rows []map[string]any
		for _, p := range resp.Organization.ProjectsV2.Nodes {
			state := "open"
			if p.Closed {
				state = "closed"
			}
			rows = append(rows, map[string]any{
				"number": p.Number,
				"title":  p.Title,
				"state":  state,
				"url":    p.URL,
			})
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		fmt.Fprintf(os.Stderr, "projects: %d\n", len(rows))
		out, _ := json.Marshal(rows)
		return printData("board.list", out)
	},
}

var boardFieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "List fields and options for a project",
	Long: `Show all custom fields on a project board with their IDs and options.
Useful for understanding what field IDs to use with 'gx items set'.

Examples:
  gx board fields --project-number 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}

		query := fmt.Sprintf(`{
			organization(login: %q) {
				projectV2(number: %d) {
					fields(first: 50) {
						nodes {
							... on ProjectV2Field {
								id
								name
								dataType
							}
							... on ProjectV2SingleSelectField {
								id
								name
								dataType
								options { id name }
							}
							... on ProjectV2IterationField {
								id
								name
								dataType
								configuration {
									iterations { id title startDate duration }
								}
							}
						}
					}
				}
			}
		}`, c.Owner(), boardProjectNum)

		data, err := c.GraphQL(context.Background(), query, nil)
		if err != nil {
			return err
		}

		var resp struct {
			Organization struct {
				ProjectV2 struct {
					Fields struct {
						Nodes []json.RawMessage `json:"nodes"`
					} `json:"fields"`
				} `json:"projectV2"`
			} `json:"organization"`
		}
		json.Unmarshal(data, &resp)

		var rows []map[string]any
		for _, raw := range resp.Organization.ProjectV2.Fields.Nodes {
			var field struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				DataType string `json:"dataType"`
				Options  []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"options"`
				Configuration *struct {
					Iterations []struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					} `json:"iterations"`
				} `json:"configuration"`
			}
			json.Unmarshal(raw, &field)
			if field.ID == "" {
				continue
			}

			var options []string
			for _, o := range field.Options {
				options = append(options, o.Name)
			}
			if field.Configuration != nil {
				for _, it := range field.Configuration.Iterations {
					options = append(options, it.Title)
				}
			}

			rows = append(rows, map[string]any{
				"id":      field.ID,
				"name":    field.Name,
				"type":    field.DataType,
				"options": strings.Join(options, ", "),
			})
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		fmt.Fprintf(os.Stderr, "fields: %d\n", len(rows))
		out, _ := json.Marshal(rows)
		return printData("board.fields", out)
	},
}
