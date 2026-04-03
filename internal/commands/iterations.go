package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	iterProjectNum int
	iterTitle      string
	iterStart      string
	iterDuration   int
)

func init() {
	rootCmd.AddCommand(iterationsCmd)
	iterationsCmd.AddCommand(iterationsListCmd)
	iterationsCmd.AddCommand(iterationsCurrentCmd)

	iterationsListCmd.Flags().IntVar(&iterProjectNum, "project-number", 0, "Project number (required)")
	iterationsListCmd.MarkFlagRequired("project-number")

	iterationsCurrentCmd.Flags().IntVar(&iterProjectNum, "project-number", 0, "Project number (required)")
	iterationsCurrentCmd.MarkFlagRequired("project-number")
}

var iterationsCmd = &cobra.Command{
	Use:     "iterations",
	Aliases: []string{"iter"},
	Short:   "Manage project iterations (sprint equivalent)",
}

var iterationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List iterations for a project",
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
							... on ProjectV2IterationField {
								id
								name
								configuration {
									iterations {
										id
										title
										startDate
										duration
									}
									completedIterations {
										id
										title
										startDate
										duration
									}
								}
							}
						}
					}
				}
			}
		}`, c.Owner(), iterProjectNum)

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
		if err := json.Unmarshal(data, &resp); err != nil {
			return err
		}

		var rows []map[string]any
		for _, raw := range resp.Organization.ProjectV2.Fields.Nodes {
			var field struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				Configuration *struct {
					Iterations []struct {
						ID        string `json:"id"`
						Title     string `json:"title"`
						StartDate string `json:"startDate"`
						Duration  int    `json:"duration"`
					} `json:"iterations"`
					CompletedIterations []struct {
						ID        string `json:"id"`
						Title     string `json:"title"`
						StartDate string `json:"startDate"`
						Duration  int    `json:"duration"`
					} `json:"completedIterations"`
				} `json:"configuration"`
			}
			json.Unmarshal(raw, &field)
			if field.Configuration == nil {
				continue
			}
			// Active iterations
			for _, it := range field.Configuration.Iterations {
				rows = append(rows, map[string]any{
					"id":         it.ID,
					"title":      it.Title,
					"start_date": it.StartDate,
					"duration":   it.Duration,
					"state":      "active",
					"field_name": field.Name,
				})
			}
			// Completed iterations
			for _, it := range field.Configuration.CompletedIterations {
				rows = append(rows, map[string]any{
					"id":         it.ID,
					"title":      it.Title,
					"start_date": it.StartDate,
					"duration":   it.Duration,
					"state":      "completed",
					"field_name": field.Name,
				})
			}
		}

		if rows == nil {
			rows = []map[string]any{}
		}
		fmt.Fprintf(os.Stderr, "iterations: %d\n", len(rows))
		out, _ := json.Marshal(rows)
		return printData("iterations.list", out)
	},
}

var iterationsCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active iteration",
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
							... on ProjectV2IterationField {
								name
								configuration {
									iterations {
										id
										title
										startDate
										duration
									}
								}
							}
						}
					}
				}
			}
		}`, c.Owner(), iterProjectNum)

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

		for _, raw := range resp.Organization.ProjectV2.Fields.Nodes {
			var field struct {
				Configuration *struct {
					Iterations []struct {
						ID        string `json:"id"`
						Title     string `json:"title"`
						StartDate string `json:"startDate"`
						Duration  int    `json:"duration"`
					} `json:"iterations"`
				} `json:"configuration"`
			}
			json.Unmarshal(raw, &field)
			if field.Configuration == nil || len(field.Configuration.Iterations) == 0 {
				continue
			}
			// First active iteration is current
			it := field.Configuration.Iterations[0]
			out, _ := json.Marshal(map[string]any{
				"id":         it.ID,
				"title":      it.Title,
				"start_date": it.StartDate,
				"duration":   it.Duration,
			})
			return printData("", out)
		}

		fmt.Fprintln(os.Stderr, "no active iteration found")
		return printData("", json.RawMessage("null"))
	},
}
