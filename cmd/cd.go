package cmd

import (
	"fmt"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/spf13/cobra"
)

var cdCmd = &cobra.Command{
	Use:   "cd <workspace-name>",
	Short: "Change directory to a workspace's clone",
	Long: `Changes your shell's current directory to the workspace's clone directory.

This command must be used with the 'cw' shell function (installed via 'cw install-shell').
It outputs a special marker that the shell integration detects and uses to change directories.

Example:
  cw cd feature-auth     # Changes to feature-auth workspace's clone directory
  cw cd                  # Interactive: select workspace from list`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var workspaceName string

		// If no args, show interactive selector
		if len(args) == 0 {
			workspaceName, err = selectWorkspaceInteractive(cfg)
			if err != nil {
				return err
			}
			if workspaceName == "" {
				return nil // User cancelled
			}
		} else {
			workspaceName = args[0]
		}

		// Get workspace
		ws, err := cfg.GetWorkspace(workspaceName)
		if err != nil {
			return fmt.Errorf("workspace '%s' not found", workspaceName)
		}

		// Get the clone path
		clonePath := ws.GetRepoPath()
		if clonePath == "" {
			return fmt.Errorf("workspace '%s' has no clone path configured", workspaceName)
		}

		// Output CD marker for shell integration to detect
		fmt.Printf("CD:%s\n", clonePath)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cdCmd)
	cdCmd.ValidArgsFunction = validWorkspaceNamesExcludeArchived
}
