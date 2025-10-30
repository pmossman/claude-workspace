package cmd

import (
	"fmt"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/session"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <workspace-name>",
	Short: "Stop a workspace and free its clone",
	Long: `Stops a workspace by killing its tmux session and freeing the associated clone.

This is useful when you want to pause work on a workspace temporarily but don't want
to archive it. The workspace remains available and can be restarted later with 'cw start'.

What this does:
- Kills the tmux session (if running)
- Frees the clone so other workspaces can use it
- Sets workspace status to 'idle'
- Preserves all workspace context files

Example:
  cw stop feature-auth       # Stop specific workspace
  cw stop                    # Interactive: select workspace to stop`,
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

		sessionMgr := session.NewManager()
		sessionName := sessionMgr.GetSessionName(workspaceName)

		// Check if session exists
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return fmt.Errorf("failed to check session: %w", err)
		}

		// Kill the tmux session if it exists
		if exists {
			fmt.Printf("Killing tmux session: %s\n", sessionName)
			if err := sessionMgr.Kill(sessionName); err != nil {
				return fmt.Errorf("failed to kill session: %w", err)
			}
		} else {
			fmt.Printf("No active tmux session for workspace '%s'\n", workspaceName)
		}

		// Free the clone if workspace is using one
		if ws.ClonePath != "" {
			if _, err := cfg.GetClone(ws.ClonePath); err == nil {
				fmt.Printf("Freeing clone: %s\n", ws.ClonePath)
				if err := cfg.FreeClone(ws.ClonePath); err != nil {
					return fmt.Errorf("failed to free clone: %w", err)
				}
			}
		}

		// Update workspace status to idle
		if err := cfg.UpdateWorkspaceStatus(workspaceName, config.StatusIdle, 0); err != nil {
			return fmt.Errorf("failed to update workspace status: %w", err)
		}

		// Save config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\n✓ Stopped workspace '%s'\n", workspaceName)
		fmt.Println("  • Tmux session killed")
		fmt.Println("  • Clone freed for other workspaces")
		fmt.Println("  • Workspace status set to idle")
		fmt.Printf("\nResume with: cw start %s\n", workspaceName)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.ValidArgsFunction = validWorkspaceNamesExcludeArchived
}
