package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/session"
	"github.com/pmossman/claude-workspace/internal/workspace"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a workspace session (interactive)",
	Long: `Starts or attaches to a tmux session for the workspace.

Interactive mode:
  cw start

Direct mode:
  cw start <workspace-name>`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var name string

		// Interactive mode if no args
		if len(args) == 0 {
			selectedName, err := interactiveWorkspaceSelect(cfg)
			if err != nil {
				return err
			}
			if selectedName == "" {
				// User cancelled
				return nil
			}
			name = selectedName
		} else {
			name = args[0]
		}

		// Get workspace
		ws, err := cfg.GetWorkspace(name)
		if err != nil {
			return err
		}

		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)
		sessionMgr := session.NewManager()

		// Get session name
		sessionName := sessionMgr.GetSessionName(name)

		// Check if session exists
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return err
		}

		// Check for existing lock (but allow reattaching to existing sessions)
		if cfg.Settings.RequireSessionLock && !exists {
			locked, pid, err := wsMgr.CheckLock(name)
			if err != nil {
				return fmt.Errorf("failed to check lock: %w", err)
			}
			if locked {
				return fmt.Errorf("workspace '%s' has an active session (PID %d)", name, pid)
			}
		}

		// If session exists, clean up any stale locks
		if exists && cfg.Settings.RequireSessionLock {
			locked, _, err := wsMgr.CheckLock(name)
			if err != nil {
				return fmt.Errorf("failed to check lock: %w", err)
			}
			if !locked {
				// Lock exists but process is dead - clean it up
				_ = wsMgr.RemoveLock(name)
			}
		}

		// Create session if it doesn't exist
		if !exists {
			fmt.Printf("Creating new session for '%s'...\n", name)
			if err := sessionMgr.Create(sessionName, ws.GetRepoPath()); err != nil {
				return err
			}

			// Customize tmux status line for this workspace
			var statusLeft string
			if ws.ClonePath != "" {
				if clone, err := cfg.GetClone(ws.ClonePath); err == nil {
					statusLeft = fmt.Sprintf("[%s] %s @ #(cd %s && git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'no-branch')",
						name, clone.RemoteName, ws.ClonePath)
				} else {
					statusLeft = fmt.Sprintf("[%s] %s", name, ws.ClonePath)
				}
			} else {
				statusLeft = fmt.Sprintf("[%s] %s", name, ws.GetRepoPath())
			}

			if err := sessionMgr.SetStatusLine(sessionName, statusLeft); err != nil {
				fmt.Printf("Warning: failed to set status line: %v\n", err)
			}

			// If auto-start is enabled, send claude command to tmux (only for new sessions)
			if cfg.Settings.AutoStartClaude {
				fmt.Println("Starting Claude Code...")
				fmt.Println()
				// Send the claude command to the tmux session
				if err := sessionMgr.SendKeys(sessionName, cfg.Settings.ClaudeCommand); err != nil {
					fmt.Printf("Warning: failed to auto-start Claude: %v\n", err)
				}
			}
		} else {
			fmt.Printf("Attaching to existing session '%s'...\n", name)
		}

		// Display header
		fmt.Println()
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Printf("  Workspace: %s\n", name)
		fmt.Printf("  Repository: %s\n", ws.GetRepoPath())

		// Display summary
		summary := wsMgr.GetSummary(name)
		if summary != "(no summary)" {
			fmt.Printf("  Summary: %s\n", summary)
		}

		// Display continuation prompt
		continuation := wsMgr.GetContinuation(name)
		if continuation != "" {
			fmt.Println("═══════════════════════════════════════════════════════════")
			fmt.Println()
			fmt.Println("📋 CONTINUATION PROMPT:")
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println(continuation)
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println()

			// Copy to clipboard if pbcopy is available (macOS)
			copyToClipboard(continuation)
		} else {
			fmt.Println("═══════════════════════════════════════════════════════════")
			fmt.Println()
			fmt.Println("(No continuation prompt yet)")
			fmt.Println()
		}

		// Create lock file
		if cfg.Settings.RequireSessionLock {
			if err := wsMgr.CreateLock(name, os.Getpid()); err != nil {
				return fmt.Errorf("failed to create lock: %w", err)
			}
		}

		// Update workspace status
		if err := cfg.UpdateWorkspaceStatus(name, config.StatusActive, os.Getpid()); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		// Show tmux tips
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("Tmux Quick Reference:")
		fmt.Println("  Ctrl-b d     - Detach (keeps Claude running)")
		fmt.Println("  Ctrl-b s     - Switch between sessions")
		fmt.Println("  cw           - Start/switch workspaces")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		// Attach to session (this will block until detach or window close)
		err = sessionMgr.Attach(sessionName)

		// Clean up lock file after detaching
		if cfg.Settings.RequireSessionLock {
			_ = wsMgr.RemoveLock(name)
		}

		// Update workspace status to idle
		_ = cfg.UpdateWorkspaceStatus(name, config.StatusIdle, 0)
		_ = cfg.Save()

		return err
	},
}

func copyToClipboard(text string) {
	// Try pbcopy (macOS)
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err == nil {
		fmt.Println("✓ Continuation prompt copied to clipboard")
		fmt.Println()
		return
	}

	// Try xclip (Linux)
	cmd = exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err == nil {
		fmt.Println("✓ Continuation prompt copied to clipboard")
		fmt.Println()
		return
	}

	// Clipboard copy not available
	fmt.Println("(Could not copy to clipboard - pbcopy/xclip not available)")
	fmt.Println()
}

// interactiveWorkspaceSelect shows fzf selector and returns selected workspace name
func interactiveWorkspaceSelect(cfg *config.Config) (string, error) {
	// Check if fzf is installed
	if err := checkFzfInstalled(); err != nil {
		return "", err
	}

	if len(cfg.Workspaces) == 0 {
		fmt.Println("No workspaces found.")
		fmt.Println("Create one with: cw create")
		return "", nil
	}

	wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)

	// Build workspace list sorted by last active
	type wsEntry struct {
		name string
		ws   *config.Workspace
	}
	var entries []wsEntry
	for name, ws := range cfg.Workspaces {
		entries = append(entries, wsEntry{name: name, ws: ws})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ws.LastActive.After(entries[j].ws.LastActive)
	})

	// Build fzf input
	var inputLines []string
	for _, entry := range entries {
		ws := entry.ws
		summary := wsMgr.GetSummary(entry.name)
		lastActive := formatTimeAgo(ws.LastActive)

		// Format: name [status] summary (time)
		line := fmt.Sprintf("%s [%s] %s (%s)",
			entry.name,
			ws.Status,
			summary,
			lastActive,
		)
		inputLines = append(inputLines, line)
	}

	input := strings.Join(inputLines, "\n")

	// Get path to self for preview command
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Build fzf command with preview
	// Use awk to extract workspace name (everything before '[')
	previewCmd := fmt.Sprintf("echo {} | awk -F'\\\\[' '{print $1}' | xargs %s preview", self)
	fzfCmd := exec.Command("fzf",
		"--ansi",
		"--no-sort",
		"--height=100%",
		"--preview="+previewCmd,
		"--preview-window=right:50%:wrap",
		"--header=Select a workspace (Ctrl-C to cancel)",
		"--prompt=Workspace> ",
	)

	// Set up pipes
	fzfCmd.Stdin = strings.NewReader(input)
	fzfCmd.Stderr = os.Stderr

	var outBuf bytes.Buffer
	fzfCmd.Stdout = &outBuf

	// Run fzf
	if err := fzfCmd.Run(); err != nil {
		// User cancelled (Ctrl-C)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 {
				return "", nil
			}
		}
		return "", fmt.Errorf("fzf failed: %w", err)
	}

	// Extract selected workspace name
	selected := strings.TrimSpace(outBuf.String())
	if selected == "" {
		return "", nil
	}

	// Parse workspace name (everything before '[')
	bracketIdx := strings.Index(selected, "[")
	if bracketIdx == -1 {
		return "", fmt.Errorf("invalid selection format")
	}
	workspaceName := strings.TrimSpace(selected[:bracketIdx])

	return workspaceName, nil
}
