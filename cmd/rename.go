package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/session"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a workspace",
	Long: `Renames a workspace by updating the config and renaming the workspace directory.
This will also update any clones that are assigned to this workspace.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		// Validate new name
		if err := config.ValidateWorkspaceName(newName); err != nil {
			return fmt.Errorf("invalid new workspace name: %w", err)
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check old workspace exists
		oldWs, err := cfg.GetWorkspace(oldName)
		if err != nil {
			return fmt.Errorf("workspace '%s' not found", oldName)
		}

		// Check new name doesn't already exist
		if _, err := cfg.GetWorkspace(newName); err == nil {
			return fmt.Errorf("workspace '%s' already exists", newName)
		}

		// Check if tmux session exists and rename it
		sessionMgr := session.NewManager()
		oldSessionName := sessionMgr.GetSessionName(oldName)
		newSessionName := sessionMgr.GetSessionName(newName)

		if exists, _ := sessionMgr.Exists(oldSessionName); exists {
			fmt.Printf("Renaming tmux session: %s -> %s\n", oldSessionName, newSessionName)
			cmd := exec.Command("tmux", "rename-session", "-t", oldSessionName, newSessionName)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to rename tmux session: %w", err)
			}
		}

		// Rename workspace directory
		oldDir := filepath.Join(cfg.Settings.WorkspaceDir, oldName)
		newDir := filepath.Join(cfg.Settings.WorkspaceDir, newName)

		if _, err := os.Stat(oldDir); err == nil {
			fmt.Printf("Renaming workspace directory: %s -> %s\n", oldDir, newDir)
			if err := os.Rename(oldDir, newDir); err != nil {
				return fmt.Errorf("failed to rename workspace directory: %w", err)
			}
		} else {
			fmt.Printf("Note: Workspace directory not found at %s\n", oldDir)
		}

		// Update workspace in config
		oldWs.Name = newName
		cfg.Workspaces[newName] = oldWs
		delete(cfg.Workspaces, oldName)

		// Update any clones that reference this workspace
		for _, clone := range cfg.Clones {
			if clone.InUseBy == oldName {
				clone.InUseBy = newName
				fmt.Printf("Updated clone at %s to reference new workspace name\n", clone.Path)
			}
		}

		// Save config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\n✓ Renamed workspace '%s' to '%s'\n", oldName, newName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
	// Only complete the first argument (old workspace name)
	renameCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete the old workspace name (first argument)
			return validWorkspaceNames(cmd, args, toComplete)
		}
		// No completion for second argument (new name)
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
