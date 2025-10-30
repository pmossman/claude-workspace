package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/workspace"
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
  cw save-context feature-auth    # Save context for specific workspace
  cw save-context                 # Interactive: select workspace`,
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

		// Show current continuation if exists
		currentCont := wsMgr.GetContinuation(workspaceName)
		if currentCont != "" {
			fmt.Println()
			fmt.Println("═══════════════════════════════════════════════════════════")
			fmt.Println("CURRENT CONTINUATION:")
			fmt.Println("───────────────────────────────────────────────────────────")
			fmt.Println(currentCont)
			fmt.Println("═══════════════════════════════════════════════════════════")
			fmt.Println()
		} else {
			fmt.Println()
			fmt.Println("No continuation currently saved.")
			fmt.Println()
		}

		// Prompt for new continuation
		fmt.Println("Enter continuation text (describe current work, what's done, what's next).")
		fmt.Println("Press Ctrl-D (EOF) when finished, or Ctrl-C to cancel.")
		fmt.Println()
		fmt.Print("Continuation:\n")

		// Read multiline input from /dev/tty
		tty, err := os.Open("/dev/tty")
		if err != nil {
			return fmt.Errorf("failed to open /dev/tty: %w", err)
		}
		defer tty.Close()

		scanner := bufio.NewScanner(tty)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		continuation := strings.TrimSpace(strings.Join(lines, "\n"))

		if continuation == "" {
			fmt.Println()
			fmt.Println("No continuation entered. Cancelled.")
			return nil
		}

		// Save continuation
		if err := wsMgr.SaveContinuation(workspaceName, continuation); err != nil {
			return fmt.Errorf("failed to save continuation: %w", err)
		}

		fmt.Println()
		fmt.Printf("✓ Saved continuation for workspace '%s'\n", workspaceName)
		fmt.Println()
		fmt.Printf("Next: Resume with 'cw start %s' or restart with 'cw restart %s'\n", workspaceName, workspaceName)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(saveContextCmd)
	saveContextCmd.ValidArgsFunction = validWorkspaceNamesExcludeArchived
}
