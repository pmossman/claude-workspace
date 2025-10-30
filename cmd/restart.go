package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/session"
	"github.com/pmossman/claude-workspace/internal/workspace"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <workspace-name>",
	Short: "Restart Claude session in a workspace",
	Long: `Restarts the Claude Code session within a workspace's tmux session.

This is useful when:
- Claude becomes unresponsive or stuck
- You want to start fresh with a new Claude session
- You need to clear Claude's context and reload with the continuation prompt

What this does:
- Sends Ctrl-C to kill the current Claude process in tmux
- Automatically starts a new Claude session
- Displays the continuation prompt (and copies to clipboard)
- Keeps the tmux session and workspace context intact

Example:
  cw restart feature-auth    # Restart specific workspace
  cw restart                 # Interactive: select workspace to restart`,
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

		// Verify workspace exists
		_, err = cfg.GetWorkspace(workspaceName)
		if err != nil {
			return fmt.Errorf("workspace '%s' not found", workspaceName)
		}

		// Prompt to save continuation before restarting
		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)
		if err := promptSaveContinuation(wsMgr, workspaceName); err != nil {
			return err
		}

		sessionMgr := session.NewManager()
		sessionName := sessionMgr.GetSessionName(workspaceName)

		// Check if session exists
		exists, err := sessionMgr.Exists(sessionName)
		if err != nil {
			return fmt.Errorf("failed to check session: %w", err)
		}

		if !exists {
			return fmt.Errorf("workspace '%s' has no active tmux session. Use 'cw start %s' instead.", workspaceName, workspaceName)
		}

		fmt.Println()
		fmt.Printf("🔄 Restarting Claude session in workspace '%s'...\n", workspaceName)
		fmt.Println()

		// Kill the Claude process directly by finding its PID
		fmt.Println("  [1/4] Finding Claude process...")

		// Find the PID of the tmux pane
		getPaneCmd := exec.Command("tmux", "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
		output, err := getPaneCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get pane PID: %w", err)
		}
		panePID := strings.TrimSpace(string(output))

		if panePID != "" {
			fmt.Printf("  [2/4] Terminating Claude process (PID: %s)...\n", panePID)

			// Kill all child processes of the tmux pane
			// Use pkill to find and kill any 'claude' processes under this pane
			killCmd := exec.Command("pkill", "-TERM", "-P", panePID, "claude")
			_ = killCmd.Run() // Ignore errors if no claude process found

			// Give it a moment to terminate gracefully
			fmt.Println("        Waiting for graceful shutdown...")
			if err := exec.Command("sleep", "0.5").Run(); err != nil {
				// Not critical if sleep fails
			}

			// Force kill if still alive
			killCmd = exec.Command("pkill", "-KILL", "-P", panePID, "claude")
			_ = killCmd.Run() // Ignore errors
			fmt.Println("        ✓ Process terminated")
		} else {
			fmt.Println("  [2/4] No active Claude process found (skipping)")
		}

		// Clear the command line
		fmt.Println("  [3/4] Clearing tmux command line...")
		if err := sessionMgr.SendKeysLiteral(sessionName, "C-c"); err != nil {
			return fmt.Errorf("failed to send Ctrl-C: %w", err)
		}
		if err := sessionMgr.SendKeysLiteral(sessionName, "C-u"); err != nil {
			return fmt.Errorf("failed to clear line: %w", err)
		}
		fmt.Println("        ✓ Command line cleared")

		// Start new Claude session
		fmt.Println("  [4/4] Starting new Claude session...")
		if err := sessionMgr.SendKeys(sessionName, cfg.Settings.ClaudeCommand); err != nil {
			return fmt.Errorf("failed to start Claude: %w", err)
		}
		fmt.Println("        ✓ Claude session started")

		// Display continuation prompt
		continuation := wsMgr.GetContinuation(workspaceName)
		if continuation != "" {
			fmt.Println()
			fmt.Println("═══════════════════════════════════════════════════════════")
			fmt.Println()
			fmt.Println("📋 CONTINUATION PROMPT:")
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println(continuation)
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println()

			// Copy to clipboard if available
			if runtime.GOOS == "darwin" {
				cmd := exec.Command("pbcopy")
				cmd.Stdin = nil
				stdin, err := cmd.StdinPipe()
				if err == nil {
					if err := cmd.Start(); err == nil {
						_, _ = stdin.Write([]byte(continuation))
						_ = stdin.Close()
						_ = cmd.Wait()
						fmt.Println("✓ Copied continuation prompt to clipboard")
					}
				}
			}
			fmt.Println()
		}

		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("✅ Successfully restarted Claude session in '%s'\n", workspaceName)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("Tip: Attach to the session with:")
		fmt.Printf("  cw start %s\n", workspaceName)

		return nil
	},
}

// promptSaveContinuation prompts the user to save continuation before restarting
func promptSaveContinuation(wsMgr *workspace.Manager, workspaceName string) error {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚠️  USER INPUT REQUIRED")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Before restarting, please update the continuation to preserve")
	fmt.Println("your current progress for the next session.")
	fmt.Println()

	// Show current continuation if exists
	currentCont := wsMgr.GetContinuation(workspaceName)
	if currentCont != "" {
		fmt.Println("CURRENT CONTINUATION:")
		fmt.Println("─────────────────────")
		fmt.Println(currentCont)
		fmt.Println("─────────────────────")
		fmt.Println()
	}

	fmt.Println("Enter new continuation (describe current work, what's done, what's next):")
	fmt.Println("  • Press Ctrl-D (EOF) when finished")
	fmt.Println("  • Press Enter on empty line to skip and keep existing continuation")
	fmt.Println()

	// Read multiline input
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		// If first line is empty, treat as skip
		if len(lines) == 0 && line == "" {
			fmt.Println("Skipping continuation update.")
			fmt.Println()
			return nil
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		// If error is EOF, that's expected
		if err.Error() != "EOF" {
			return fmt.Errorf("error reading input: %w", err)
		}
	}

	continuation := strings.TrimSpace(strings.Join(lines, "\n"))

	if continuation == "" {
		fmt.Println()
		fmt.Println("⏭️  Skipped - keeping existing continuation")
		fmt.Println()
		return nil
	}

	// Save continuation
	if err := wsMgr.SaveContinuation(workspaceName, continuation); err != nil {
		return fmt.Errorf("failed to save continuation: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Saved continuation for workspace '%s'\n", workspaceName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(restartCmd)
	restartCmd.ValidArgsFunction = validWorkspaceNamesExcludeArchived
}
