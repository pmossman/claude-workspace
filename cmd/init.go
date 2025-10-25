package cmd

import (
	"fmt"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/session"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize claude-workspace configuration",
	Long:  `Creates the configuration directory and initial config file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if tmux is installed
		sessionMgr := session.NewManager()
		if err := sessionMgr.CheckTmuxInstalled(); err != nil {
			return err
		}

		// Load or create default config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Save config to ensure directory structure exists
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("✓ Initialized claude-workspace")
		fmt.Printf("  Config directory: %s\n", cfg.Settings.WorkspaceDir)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Install shell integration: cw install-shell")
		fmt.Println("  2. Add a remote: cw add-remote <name> <git-url> --clone-dir <path>")
		fmt.Println("  3. Create a workspace: cw create <name> --remote <remote-name>")
		fmt.Println("  Or use the interactive selector: cw")

		return nil
	},
}
