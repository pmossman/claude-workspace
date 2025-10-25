package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claude-workspace",
	Short: "Manage Claude Code workspaces with context preservation",
	Long: `claude-workspace is a tool for managing multiple Claude Code sessions across
different repository clones, with automatic context preservation and session management.`,
	RunE: selectCmd.RunE, // Default to interactive selector
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Register subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(installShellCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(selectCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(quickCmd)

	// Remote and clone management
	rootCmd.AddCommand(addRemoteCmd)
	rootCmd.AddCommand(listRemotesCmd)
	rootCmd.AddCommand(newCloneCmd)
	rootCmd.AddCommand(importCloneCmd)
	rootCmd.AddCommand(clonesCmd)
}
