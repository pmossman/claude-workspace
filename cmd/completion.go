package cmd

import (
	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/spf13/cobra"
)

// validWorkspaceNames returns a list of valid workspace names for completion
func validWorkspaceNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Collect workspace names
	var names []string
	for name := range cfg.Workspaces {
		names = append(names, name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// validWorkspaceNamesExcludeArchived returns non-archived workspace names for completion
func validWorkspaceNamesExcludeArchived(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Collect non-archived workspace names
	var names []string
	for name, ws := range cfg.Workspaces {
		if ws.Status != config.StatusArchived {
			names = append(names, name)
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// validRemoteNames returns a list of valid remote names for completion
func validRemoteNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Collect remote names
	var names []string
	for name := range cfg.Remotes {
		names = append(names, name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}
