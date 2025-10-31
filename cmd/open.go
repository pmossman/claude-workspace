package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/pmossman/claudew/internal/config"
	"github.com/pmossman/claudew/internal/workspace"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <workspace-name>",
	Short: "Open workspace directory in file browser",
	Long: `Opens the workspace directory in your system's default file browser.
This lets you view and edit the workspace's markdown files (context.md, decisions.md, etc.) directly.

On macOS: Opens in Finder
On Linux: Uses xdg-open
On Windows: Uses explorer`,
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
		if _, err := cfg.GetWorkspace(workspaceName); err != nil {
			return fmt.Errorf("workspace '%s' not found", workspaceName)
		}

		// Get workspace directory
		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)
		workspaceDir := wsMgr.GetPath(workspaceName)

		// Open in file browser based on OS
		var openCmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			openCmd = exec.Command("open", workspaceDir)
		case "linux":
			openCmd = exec.Command("xdg-open", workspaceDir)
		case "windows":
			openCmd = exec.Command("explorer", workspaceDir)
		default:
			return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}

		if err := openCmd.Run(); err != nil {
			return fmt.Errorf("failed to open workspace directory: %w", err)
		}

		fmt.Printf("✓ Opened workspace directory: %s\n", workspaceDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.ValidArgsFunction = validWorkspaceNamesExcludeArchived
}
