package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pmossman/claudew/internal/config"
	"github.com/spf13/cobra"
)

var addRemoteCmd = &cobra.Command{
	Use:   "add-remote [name] [git-url] [--clone-dir <path>]",
	Short: "Register a remote repository",
	Long: `Registers a remote repository for clone management.
The clone-dir is where new clones will be created (e.g., ~/dev/airbyte-clones).

If called without arguments, runs interactively.`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name, url, cloneDir string

		// Interactive mode if no args provided
		if len(args) == 0 {
			// Reopen /dev/tty for both reading and writing to ensure prompts are visible after fzf
			tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
			if err != nil {
				return fmt.Errorf("failed to open terminal: %w", err)
			}
			defer tty.Close()

			fmt.Fprintln(tty)
			fmt.Fprint(tty, "Remote name (e.g., 'my-app'): ")
			fmt.Fscanln(tty, &name)
			if name == "" {
				return fmt.Errorf("remote name is required")
			}

			fmt.Fprint(tty, "Git URL (e.g., 'git@github.com:org/repo.git'): ")
			fmt.Fscanln(tty, &url)
			if url == "" {
				return fmt.Errorf("git URL is required")
			}

			fmt.Fprint(tty, "Clone directory (e.g., '~/dev/my-app-clones'): ")
			fmt.Fscanln(tty, &cloneDir)
			if cloneDir == "" {
				return fmt.Errorf("clone directory is required")
			}
		} else if len(args) == 2 {
			name = args[0]
			url = args[1]
			cloneDir, _ = cmd.Flags().GetString("clone-dir")
			if cloneDir == "" {
				return fmt.Errorf("--clone-dir is required")
			}
		} else {
			return fmt.Errorf("provide either no arguments (interactive) or both name and URL with --clone-dir flag")
		}

		// Expand ~ in path
		if len(cloneDir) >= 2 && cloneDir[:2] == "~/" {
			home, _ := os.UserHomeDir()
			cloneDir = filepath.Join(home, cloneDir[2:])
		} else if cloneDir == "~" {
			cloneDir, _ = os.UserHomeDir()
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
		fmt.Println()
		fmt.Println("Next: Create a workspace for this remote")
		fmt.Println("  Run 'claudew' to open the interactive menu")

		return nil
	},
}

func init() {
	addRemoteCmd.Flags().String("clone-dir", "", "Base directory for clones (required)")
	addRemoteCmd.MarkFlagRequired("clone-dir")
}
