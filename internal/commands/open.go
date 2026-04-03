package commands

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var openURLOnly bool

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.Flags().BoolVar(&openURLOnly, "url", false, "Print URL only")
}

var openCmd = &cobra.Command{
	Use:   "open <issue-number>",
	Short: "Open an issue in the browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := getClient(cmd)
		if err != nil {
			return err
		}
		url := fmt.Sprintf("https://github.com/%s/%s/issues/%s", c.Owner(), c.Repo(), args[0])
		if openURLOnly {
			fmt.Fprintln(cmd.OutOrStdout(), url)
			return nil
		}
		var openCmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			openCmd = exec.Command("xdg-open", url)
		case "darwin":
			openCmd = exec.Command("open", url)
		default:
			fmt.Fprintln(cmd.OutOrStdout(), url)
			return nil
		}
		openCmd.Start()
		return nil
	},
}
