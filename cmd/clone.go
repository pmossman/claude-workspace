package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/template"
	"github.com/pmossman/claude-workspace/internal/workspace"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <from-name> <to-name> <repo-path>",
	Short: "Clone a workspace to a new one",
	Long: `Creates a new workspace by copying context from an existing workspace.
Useful when branching work to a new feature from an existing workspace.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromName := args[0]
		toName := args[1]
		repoPath := args[2]

		// Expand ~ in path
		if repoPath[:2] == "~/" {
			home, _ := os.UserHomeDir()
			repoPath = filepath.Join(home, repoPath[2:])
		}

		// Make path absolute
		absRepoPath, err := filepath.Abs(repoPath)
		if err != nil {
			return fmt.Errorf("invalid repo path: %w", err)
		}

		// Check if repo path exists
		if _, err := os.Stat(absRepoPath); os.IsNotExist(err) {
			return fmt.Errorf("repo path does not exist: %s", absRepoPath)
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check source workspace exists
		if _, err := cfg.GetWorkspace(fromName); err != nil {
			return fmt.Errorf("source workspace '%s' not found", fromName)
		}

		// Add new workspace to config
		if err := cfg.AddWorkspace(toName, absRepoPath); err != nil {
			return err
		}

		// Clone workspace directory
		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)
		if err := wsMgr.Clone(fromName, toName); err != nil {
			return err
		}

		// Generate CLAUDE.md in new repo
		workspaceDir := wsMgr.GetPath(toName)
		if err := template.GenerateClaudeMd(toName, workspaceDir, absRepoPath); err != nil {
			return err
		}

		// Ensure .gitignore has .claude/
		if err := template.EnsureGitignore(absRepoPath); err != nil {
			return err
		}

		// Save config
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Cloned workspace '%s' → '%s'\n", fromName, toName)
		fmt.Printf("  Repository: %s\n", absRepoPath)
		fmt.Printf("  Workspace dir: %s\n", workspaceDir)
		fmt.Println("\nContext files copied from source workspace.")
		fmt.Println("Next: claude-workspace start", toName)

		return nil
	},
}
