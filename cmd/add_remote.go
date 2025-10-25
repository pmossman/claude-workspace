package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/spf13/cobra"
)

var addRemoteCmd = &cobra.Command{
	Use:   "add-remote <name> <git-url> --clone-dir <path>",
	Short: "Register a remote repository",
	Long: `Registers a remote repository for clone management.
The clone-dir is where new clones will be created (e.g., ~/dev/airbyte-clones).`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := args[1]

		cloneDir, _ := cmd.Flags().GetString("clone-dir")
		if cloneDir == "" {
			return fmt.Errorf("--clone-dir is required")
		}

		// Expand ~ in path
		if cloneDir[:2] == "~/" {
			home, _ := os.UserHomeDir()
			cloneDir = filepath.Join(home, cloneDir[2:])
		}

		// Make path absolute
		absCloneDir, err := filepath.Abs(cloneDir)
		if err != nil {
			return fmt.Errorf("invalid clone-dir path: %w", err)
		}

		// Create directory if it doesn't exist
		if err := os.MkdirAll(absCloneDir, 0755); err != nil {
			return fmt.Errorf("failed to create clone directory: %w", err)
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Add remote
		if err := cfg.AddRemote(name, url, absCloneDir); err != nil {
			return err
		}

		// Save config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Added remote '%s'\n", name)
		fmt.Printf("  URL: %s\n", url)
		fmt.Printf("  Clone directory: %s\n", absCloneDir)
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Import existing clones: claude-workspace import-clone %s <path>\n", name)
		fmt.Printf("  2. Or create a workspace: claude-workspace create <workspace-name> --remote %s\n", name)

		return nil
	},
}

func init() {
	addRemoteCmd.Flags().String("clone-dir", "", "Base directory for clones (required)")
	addRemoteCmd.MarkFlagRequired("clone-dir")
}
