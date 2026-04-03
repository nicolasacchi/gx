package commands

import (
	"encoding/json"
	"fmt"

	"github.com/nicolasacchi/gx/internal/config"
	"github.com/spf13/cobra"
)

var (
	configOwner string
	configRepo  string
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configCurrentCmd)

	configAddCmd.Flags().StringVar(&configOwner, "owner", "", "GitHub org/user (required)")
	configAddCmd.Flags().StringVar(&configRepo, "repo", "", "Repository name (required)")
	configAddCmd.MarkFlagRequired("owner")
	configAddCmd.MarkFlagRequired("repo")
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gx configuration",
}

var configAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a named project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.AddProject(args[0], configOwner, configRepo); err != nil {
			return err
		}
		if !quietFlag {
			fmt.Fprintf(cmd.OutOrStdout(), "Project %q saved\n", args[0])
		}
		return nil
	},
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a named project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.RemoveProject(args[0])
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.ListProjects()
		if err != nil {
			return fmt.Errorf("no config file; run 'gx config add' first")
		}
		var rows []map[string]any
		for name, p := range cfg.Projects {
			isDefault := "no"
			if name == cfg.DefaultProject {
				isDefault = "yes"
			}
			rows = append(rows, map[string]any{
				"name":    name,
				"owner":   p.Owner,
				"repo":    p.Repo,
				"default": isDefault,
			})
		}
		data, _ := json.Marshal(rows)
		return printData("config.list", data)
	},
}

var configUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set default project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.SetDefaultProject(args[0])
	},
}

var configCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current default project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.ListProjects()
		if err != nil {
			return fmt.Errorf("no config file")
		}
		fmt.Fprintln(cmd.OutOrStdout(), cfg.DefaultProject)
		return nil
	},
}
