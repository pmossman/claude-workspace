package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/git"
	"github.com/spf13/cobra"
)

var importCloneCmd = &cobra.Command{
	Use:   "import-clone <remote-name> <clone-path>",
	Short: "Import an existing clone into management",
	Long:  `Registers an existing repository clone for workspace management.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName := args[0]
		clonePath := args[1]

		// Expand ~ in path
		if clonePath[:2] == "~/" {
			home, _ := os.UserHomeDir()
			clonePath = filepath.Join(home, clonePath[2:])
		}

		// Make path absolute
		absClonePath, err := filepath.Abs(clonePath)
		if err != nil {
			return fmt.Errorf("invalid clone path: %w", err)
		}

		// Check if path exists
		if _, err := os.Stat(absClonePath); os.IsNotExist(err) {
			return fmt.Errorf("clone path does not exist: %s", absClonePath)
		}

		// Check if it's a git repo
		if !git.IsGitRepo(absClonePath) {
			return fmt.Errorf("path is not a git repository: %s", absClonePath)
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Get remote
		remote, err := cfg.GetRemote(remoteName)
		if err != nil {
			return err
		}

		// Verify remote URL matches
		repoURL, err := git.GetRemoteURL(absClonePath)
		if err != nil {
			fmt.Printf("Warning: Could not verify remote URL: %v\n", err)
		} else if repoURL != remote.URL {
			fmt.Printf("Warning: Clone's remote URL (%s) doesn't match registered remote URL (%s)\n", repoURL, remote.URL)
			fmt.Print("Continue anyway? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				return fmt.Errorf("import cancelled")
			}
		}

		// Add clone to config
		if err := cfg.AddClone(absClonePath, remoteName); err != nil {
			return err
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(absClonePath)
		if err != nil {
			branch = "unknown"
		}

		clone, _ := cfg.GetClone(absClonePath)
		clone.CurrentBranch = branch

		// Save config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Imported clone at %s\n", absClonePath)
		fmt.Printf("  Remote: %s\n", remoteName)
		fmt.Printf("  Branch: %s\n", branch)
		fmt.Printf("  Status: Free (available for workspaces)\n")

		return nil
	},
}
