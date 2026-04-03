package commands

import (
	"errors"
	"os"

	"github.com/nicolasacchi/gx/internal/client"
	"github.com/nicolasacchi/gx/internal/config"
	"github.com/nicolasacchi/gx/internal/output"
	"github.com/spf13/cobra"
)

var (
	version     = "dev"
	tokenFlag   string
	ownerFlag   string
	repoFlag    string
	projectFlag string
	jsonFlag    bool
	jqFlag      string
	limitFlag   int
	verboseFlag bool
	quietFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "gx",
	Short: "gx — GitHub Explorer CLI for Projects, Sub-issues, Iterations",
	Long: `gx is a purpose-built CLI for GitHub Projects management.
Handles sub-issues, iterations (sprints), milestones (epics), and project board
field updates — everything gh CLI can't do.

Usage examples:
  gx issues list --label "type:bug" --state open
  gx sub-issues add 123 456
  gx milestones create --title "v2.1" --due "2026-06-01"
  gx items set 123 --project-number 1 --status "In Progress"
  gx iterations list --project-number 1
  gx overview`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) {
			output.PrintError(apiErr.Error(), apiErr.StatusCode)
			os.Exit(apiErr.ExitCode())
		}
	}
	return err
}

func getClient(cmd *cobra.Command) (*client.Client, error) {
	creds, err := config.LoadCredentials(tokenFlag, ownerFlag, repoFlag, projectFlag)
	if err != nil {
		return nil, err
	}
	return client.New(creds.Token, creds.Owner, creds.Repo, verboseFlag), nil
}

func isJSONMode() bool {
	return output.IsJSON(jsonFlag, jqFlag)
}

func printData(command string, data []byte) error {
	return output.PrintData(command, data, isJSONMode(), jqFlag)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&tokenFlag, "token", "", "GitHub token (overrides GITHUB_TOKEN and gh auth)")
	rootCmd.PersistentFlags().StringVar(&ownerFlag, "owner", "", "GitHub org/user (overrides GX_OWNER)")
	rootCmd.PersistentFlags().StringVar(&repoFlag, "repo", "", "Repository name (overrides GX_REPO)")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Named project from config file")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Force JSON output (auto-enabled when piped)")
	rootCmd.PersistentFlags().StringVar(&jqFlag, "jq", "", "Apply gjson path filter to JSON output")
	rootCmd.PersistentFlags().IntVar(&limitFlag, "limit", 50, "Max results")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Print request/response details to stderr")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress non-error output")
}
