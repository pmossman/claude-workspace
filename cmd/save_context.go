package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pmossman/claudew/internal/config"
	"github.com/pmossman/claudew/internal/workspace"
	"github.com/spf13/cobra"
)

var saveContextCmd = &cobra.Command{
	Use:   "save-context <workspace-name>",
	Short: "Save context and continuation for a workspace",
	Long: `Interactively prompts to update the continuation.md file for a workspace.

This is useful:
- Before restarting Claude to preserve progress
- At natural stopping points during work
- When switching between workspaces
- To manually ensure context is saved

The command will prompt you to describe:
- What you're currently working on
- What has been completed
- What should be done next

Example:
  claudew save-context feature-auth    # Save context for specific workspace
  claudew save-context                 # Interactive: select workspace`,
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

		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)

		// Reopen /dev/tty for both reading and writing to ensure output is visible after fzf
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return fmt.Errorf("failed to open terminal: %w", err)
		}
		defer tty.Close()

		fmt.Fprintln(tty)

		// Show current continuation if exists
		currentCont := wsMgr.GetContinuation(workspaceName)
		if currentCont != "" {
			fmt.Fprintln(tty, "Current continuation:")
			fmt.Fprintln(tty, currentCont)
			fmt.Fprintln(tty)
		}

		// Prompt for new continuation
		fmt.Fprintln(tty, "Enter continuation text (describe current work, what's done, what's next).")
		fmt.Fprintln(tty, "Press Ctrl-D when finished, or Enter on empty line to keep current.")
		fmt.Fprintln(tty)
		fmt.Fprint(tty, "> ")

		// Read from the same tty
		scanner := bufio.NewScanner(tty)
		var lines []string
		for scanner.Scan() {
			line := scanner.Text()
			// If first line is empty, keep existing continuation
			if len(lines) == 0 && line == "" {
				fmt.Fprintln(tty, "Keeping existing continuation.")
				fmt.Fprintln(tty)
				return nil
			}
			lines = append(lines, line)
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		continuation := strings.TrimSpace(strings.Join(lines, "\n"))

		if continuation == "" {
			fmt.Fprintln(tty)
			fmt.Fprintln(tty, "Keeping existing continuation.")
			return nil
		}

		// Save continuation
		if err := wsMgr.SaveContinuation(workspaceName, continuation); err != nil {
			return fmt.Errorf("failed to save continuation: %w", err)
		}

		fmt.Println()
		fmt.Printf("✓ Saved continuation for workspace '%s'\n", workspaceName)
		fmt.Println()
		fmt.Printf("Next: Resume with 'claudew start %s' or restart with 'claudew restart %s'\n", workspaceName, workspaceName)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(saveContextCmd)
	saveContextCmd.ValidArgsFunction = validWorkspaceNamesExcludeArchived
}
