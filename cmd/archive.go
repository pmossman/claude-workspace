package cmd

import (
	"fmt"

	"github.com/pmossman/claudew/internal/config"
	"github.com/pmossman/claudew/internal/template"
	"github.com/pmossman/claudew/internal/workspace"
	"github.com/spf13/cobra"
)

var archiveCmd = &cobra.Command{
	Use:   "archive <name>",
	Short: "Archive a workspace",
	Long:  `Archives a workspace by moving its directory and updating its status.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Get workspace
		ws, err := cfg.GetWorkspace(name)
		if err != nil {
			return err
		}

		// Check if workspace is active
		if ws.Status == config.StatusActive {
			return fmt.Errorf("cannot archive active workspace '%s'. Stop the session first.", name)
		}

		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)

		// Archive workspace directory
		if err := wsMgr.Archive(name); err != nil {
			return err
		}

		// Remove CLAUDE.md from repo
		if err := template.RemoveClaudeMd(ws.GetRepoPath()); err != nil {
			fmt.Printf("Warning: failed to remove CLAUDE.md: %v\n", err)
		}

		// Free the clone if it's managed
		if ws.ClonePath != "" {
			if err := cfg.FreeClone(ws.ClonePath); err != nil {
				fmt.Printf("Warning: failed to free clone: %v\n", err)
			} else {
				fmt.Printf("  Clone freed: %s\n", ws.ClonePath)
			}
		}

		// Update status and save
		if err := cfg.UpdateWorkspaceStatus(name, config.StatusArchived, 0); err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Archived workspace '%s'\n", name)

		return nil
	},
}

func init() {
	archiveCmd.ValidArgsFunction = validWorkspaceNamesExcludeArchived
}
