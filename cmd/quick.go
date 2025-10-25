package cmd

import (
	"fmt"
	"os"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/session"
	"github.com/spf13/cobra"
)

var quickCmd = &cobra.Command{
	Use:   "quick",
	Short: "Start a quick floating session (no workspace)",
	Long: `Starts a tmux session without workspace context management.
Useful for quick questions or tasks that don't need long-term context preservation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config for Claude command
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		sessionMgr := session.NewManager()
		sessionName := "claude-quick"

		// Get current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Check if session exists
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return err
		}

		// Create session if it doesn't exist
		if !exists {
			fmt.Println("Creating quick session...")
			if err := sessionMgr.Create(sessionName, cwd); err != nil {
				return err
			}
		} else {
			fmt.Println("Attaching to existing quick session...")
		}

		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println("  Quick Session (No Workspace)")
		fmt.Println("  No context preservation - for quick tasks only")
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Println()

		// Auto-start Claude if enabled
		if cfg.Settings.AutoStartClaude && !exists {
			fmt.Println("Starting Claude Code...")
			fmt.Println()
			if err := sessionMgr.SendKeys(sessionName, cfg.Settings.ClaudeCommand); err != nil {
				fmt.Printf("Warning: failed to auto-start Claude: %v\n", err)
			}
		}

		// Attach to session
		return sessionMgr.Attach(sessionName)
	},
}
