package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const shellIntegration = `
# claude-workspace shell integration
cw() {
  local output
  output=$(claude-workspace "$@" 2>&1)
  local exit_code=$?

  # Check if output starts with CD: marker (for clone navigation)
  if echo "$output" | grep -q "^CD:"; then
    local clone_path=$(echo "$output" | grep "^CD:" | cut -d: -f2-)
    if [ -n "$clone_path" ] && [ -d "$clone_path" ]; then
      cd "$clone_path" || return 1
      echo "📂 Changed to: $clone_path"
      return 0
    fi
  fi

  # Otherwise, just display the output normally
  echo "$output"
  return $exit_code
}
`

var installShellCmd = &cobra.Command{
	Use:   "install-shell",
	Short: "Install shell integration (adds cw function to your shell)",
	Long: `Installs shell integration for interactive features.

Adds the 'cw' function to your ~/.zshrc or ~/.bashrc:
  cw - Interactive super-prompt with workspace management and clone navigation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		// Detect shell
		shell := os.Getenv("SHELL")
		var rcFile string
		if strings.Contains(shell, "zsh") {
			rcFile = filepath.Join(home, ".zshrc")
		} else if strings.Contains(shell, "bash") {
			rcFile = filepath.Join(home, ".bashrc")
		} else {
			return fmt.Errorf("unsupported shell: %s (only bash and zsh supported)", shell)
		}

		// Check if already installed
		content, err := os.ReadFile(rcFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", rcFile, err)
		}

		if strings.Contains(string(content), "# claude-workspace shell integration") {
			fmt.Println("✓ Shell integration already installed")
			fmt.Printf("  Location: %s\n", rcFile)
			fmt.Println("\nAvailable commands:")
			fmt.Println("  cw              - Interactive super-prompt (workspaces, clones, actions)")
			fmt.Println("  cw start <name> - Start a workspace")
			fmt.Println("  cw create       - Create a workspace")
			return nil
		}

		// Append integration
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", rcFile, err)
		}
		defer f.Close()

		if _, err := f.WriteString(shellIntegration); err != nil {
			return fmt.Errorf("failed to write to %s: %w", rcFile, err)
		}

		fmt.Println("✓ Shell integration installed")
		fmt.Printf("  Location: %s\n", rcFile)
		fmt.Println("\nAvailable commands:")
		fmt.Println("  cw              - Interactive super-prompt (workspaces, clones, actions)")
		fmt.Println("  cw start <name> - Start a workspace")
		fmt.Println("  cw create       - Create a workspace")
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  ACTION REQUIRED: Activate shell integration")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("Run this command now:")
		fmt.Printf("  source %s\n", rcFile)
		fmt.Println()
		fmt.Println("Or restart your terminal")
		fmt.Println()

		return nil
	},
}
