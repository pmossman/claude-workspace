package cmd

import (
	"fmt"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/workspace"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a workspace",
	Long:  `Displays detailed information including context, decisions, and continuation prompt.`,
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

		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)

		// Display workspace info
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Printf("Workspace: %s\n", name)
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Printf("Status:       %s\n", formatStatus(ws.Status))
		fmt.Printf("Repository:   %s\n", ws.GetRepoPath())

		// Show clone info if managed
		if ws.ClonePath != "" {
			if clone, err := cfg.GetClone(ws.ClonePath); err == nil {
				fmt.Printf("Remote:       %s\n", clone.RemoteName)
				fmt.Printf("Branch:       %s\n", clone.CurrentBranch)
			}
		}

		fmt.Printf("Created:      %s\n", ws.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Last Active:  %s (%s)\n", ws.LastActive.Format("2006-01-02 15:04:05"), formatTimeAgo(ws.LastActive))

		summary := wsMgr.GetSummary(name)
		if summary != "(no summary)" {
			fmt.Printf("Summary:      %s\n", summary)
		}

		if ws.SessionPID > 0 {
			fmt.Printf("Session PID:  %d\n", ws.SessionPID)
		}

		// Display continuation
		continuation := wsMgr.GetContinuation(name)
		if continuation != "" {
			fmt.Println()
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println("CONTINUATION PROMPT:")
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println(continuation)
		}

		// Display context preview
		context := wsMgr.GetContext(name)
		if context != "(no context yet)" {
			fmt.Println()
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println("CONTEXT (preview):")
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println(context)
		}

		fmt.Println()
		fmt.Printf("Workspace directory: %s\n", wsMgr.GetPath(name))

		return nil
	},
}

func init() {
	infoCmd.ValidArgsFunction = validWorkspaceNames
}
