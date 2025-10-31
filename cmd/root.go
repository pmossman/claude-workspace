package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "claudew",
	Short: "Manage Claude Code workspaces with context preservation",
	Long: `claudew is a tool for managing multiple Claude Code sessions across
different repository clones, with automatic context preservation and session management.

The shell function 'claudew' wraps this binary and adds directory navigation features.
Install it with: claudew install-shell`,
	RunE: selectCmd.RunE, // Default to interactive selector
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Disable standalone completion command (integrated into install-shell)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Register subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(installShellCmd)
	rootCmd.AddCommand(uninstallShellCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(selectCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(forkCmd)

	// Remote and clone management
	rootCmd.AddCommand(addRemoteCmd)
	rootCmd.AddCommand(listRemotesCmd)
	rootCmd.AddCommand(newCloneCmd)
	rootCmd.AddCommand(clonesCmd)
}
