package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var updateCompletionCmd = &cobra.Command{
	Use:    "update-completion",
	Short:  "Regenerate shell completion scripts",
	Hidden: true, // Hidden from main help, but available
	Long: `Regenerates the shell completion scripts with the latest command definitions.
Run this after updating claude-workspace to get completion for new commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		shell := os.Getenv("SHELL")
		var completionPath string
		var completionScript strings.Builder

		if strings.Contains(shell, "zsh") {
			// Generate zsh completion
			completionDir := filepath.Join(home, ".zsh", "completion")
			if err := os.MkdirAll(completionDir, 0755); err != nil {
				return fmt.Errorf("failed to create completion directory: %w", err)
			}
			completionPath = filepath.Join(completionDir, "_claude-workspace")

			if err := rootCmd.GenZshCompletion(&completionScript); err != nil {
				return fmt.Errorf("failed to generate zsh completion: %w", err)
			}
		} else if strings.Contains(shell, "bash") {
			// Generate bash completion
			completionPath = filepath.Join(home, ".claude-workspace-completion.bash")

			if err := rootCmd.GenBashCompletion(&completionScript); err != nil {
				return fmt.Errorf("failed to generate bash completion: %w", err)
			}
		} else {
			return fmt.Errorf("unsupported shell: %s (only bash and zsh supported)", shell)
		}

		// Write completion script
		if err := os.WriteFile(completionPath, []byte(completionScript.String()), 0644); err != nil {
			return fmt.Errorf("failed to write completion script: %w", err)
		}

		fmt.Println("✓ Completion script regenerated")
		fmt.Printf("  Location: %s\n", completionPath)
		fmt.Println("\nTo activate:")
		if strings.Contains(shell, "zsh") {
			fmt.Println("  exec zsh")
		} else {
			fmt.Printf("  source %s\n", completionPath)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCompletionCmd)
}
