package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/pmossman/claudew/internal/config"
	"github.com/pmossman/claudew/internal/git"
	"github.com/spf13/cobra"
)

var newCloneCmd = &cobra.Command{
	Use:   "new-clone <remote-name>",
	Short: "Create a new clone of a remote repository",
	Long:  `Clones the remote repository to a new numbered directory in the clone base directory.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName := args[0]

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

		// Get next clone number
		cloneNum := cfg.GetNextCloneNumber(remoteName)
		clonePath := filepath.Join(remote.CloneBaseDir, fmt.Sprintf("%d", cloneNum))

		fmt.Printf("Creating clone %d of '%s'...\n", cloneNum, remoteName)
		fmt.Printf("  Cloning from: %s\n", remote.URL)
		fmt.Printf("  To: %s\n", clonePath)
		fmt.Println()

		// Clone the repository
		if err := git.Clone(remote.URL, clonePath); err != nil {
			return err
		}

		// Add clone to config
		if err := cfg.AddClone(clonePath, remoteName); err != nil {
			return err
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(clonePath)
		if err != nil {
			branch = "unknown"
		}

		clone, _ := cfg.GetClone(clonePath)
		clone.CurrentBranch = branch

		// Save config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Created clone at %s\n", clonePath)
		fmt.Printf("  Branch: %s\n", branch)
		fmt.Printf("  Status: Free (available for workspaces)\n")

		return nil
	},
}
