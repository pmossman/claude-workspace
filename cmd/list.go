package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/pmossman/claudew/internal/config"
	"github.com/pmossman/claudew/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	listArchived bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	Long:  `Lists all workspaces with their status and last active time.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.Workspaces) == 0 {
			fmt.Println("No workspaces found.")
			fmt.Println("Create one with: claudew create <name> <repo-path>")
			return nil
		}

		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)

		// Sort workspaces by last active (most recent first)
		type wsEntry struct {
			name string
			ws   *config.Workspace
		}
		var entries []wsEntry
		for name, ws := range cfg.Workspaces {
			// Skip archived workspaces unless explicitly requested
			if !listArchived && ws.Status == config.StatusArchived {
				continue
			}
			entries = append(entries, wsEntry{name: name, ws: ws})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].ws.LastActive.After(entries[j].ws.LastActive)
		})

		// Print header
		fmt.Printf("%-20s %-10s %-50s %s\n", "NAME", "STATUS", "REPO PATH", "LAST ACTIVE")
		fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────")

		// Print workspaces
		for _, entry := range entries {
			ws := entry.ws
			summary := wsMgr.GetSummary(entry.name)

			// Format last active time
			lastActive := formatTimeAgo(ws.LastActive)

			// Status with color codes
			statusStr := formatStatus(ws.Status)

			// Truncate repo path if too long
			repoPath := ws.GetRepoPath()
			if len(repoPath) > 50 {
				repoPath = "..." + repoPath[len(repoPath)-47:]
			}

			fmt.Printf("%-20s %-10s %-50s %s\n", entry.name, statusStr, repoPath, lastActive)

			// Print summary and clone info
			if summary != "(no summary)" {
				fmt.Printf("  └─ %s", summary)

				// Add clone info if managed
				if ws.ClonePath != "" {
					if clone, err := cfg.GetClone(ws.ClonePath); err == nil {
						fmt.Printf(" (%s, %s)", clone.RemoteName, clone.CurrentBranch)
					}
				} else {
					fmt.Printf(" [unmanaged]")
				}
				fmt.Println()
			} else if ws.ClonePath != "" {
				// Show clone info even without summary
				if clone, err := cfg.GetClone(ws.ClonePath); err == nil {
					fmt.Printf("  └─ (%s, %s)\n", clone.RemoteName, clone.CurrentBranch)
				}
			} else {
				// No summary and no clone - show unmanaged
				fmt.Printf("  └─ [unmanaged]\n")
			}
		}

		return nil
	},
}

func formatStatus(status string) string {
	switch status {
	case config.StatusActive:
		return "[active]"
	case config.StatusIdle:
		return "[idle]"
	case config.StatusArchived:
		return "[archived]"
	default:
		return status
	}
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

func init() {
	listCmd.Flags().BoolVar(&listArchived, "archived", false, "Include archived workspaces in the list")
}
