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

		// Generate and install completion
		var completionScript string
		var completionPath string
		if strings.Contains(shell, "zsh") {
			// Generate zsh completion
			completionDir := filepath.Join(home, ".zsh", "completion")
			if err := os.MkdirAll(completionDir, 0755); err != nil {
				return fmt.Errorf("failed to create completion directory: %w", err)
			}
			completionPath = filepath.Join(completionDir, "_claude-workspace")

			// Generate completion script to string
			var builder strings.Builder
			if err := rootCmd.GenZshCompletion(&builder); err != nil {
				return fmt.Errorf("failed to generate zsh completion: %w", err)
			}
			completionScript = builder.String()
		} else {
			// Generate bash completion
			completionPath = filepath.Join(home, ".claude-workspace-completion.bash")

			// Generate completion script to string
			var builder strings.Builder
			if err := rootCmd.GenBashCompletion(&builder); err != nil {
				return fmt.Errorf("failed to generate bash completion: %w", err)
			}
			completionScript = builder.String()
		}

		// Write completion script
		if err := os.WriteFile(completionPath, []byte(completionScript), 0644); err != nil {
			return fmt.Errorf("failed to write completion script: %w", err)
		}

		// Append integration and completion sourcing
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", rcFile, err)
		}
		defer f.Close()

		// Write shell integration
		if _, err := f.WriteString(shellIntegration); err != nil {
			return fmt.Errorf("failed to write to %s: %w", rcFile, err)
		}

		// Add completion sourcing
		if strings.Contains(shell, "zsh") {
			completionSetup := fmt.Sprintf(`
# claude-workspace completion
fpath=(~/.zsh/completion $fpath)
autoload -Uz compinit && compinit
`)
			if _, err := f.WriteString(completionSetup); err != nil {
				return fmt.Errorf("failed to write completion setup: %w", err)
			}
		} else {
			completionSetup := fmt.Sprintf(`
# claude-workspace completion
source %s
`, completionPath)
			if _, err := f.WriteString(completionSetup); err != nil {
				return fmt.Errorf("failed to write completion setup: %w", err)
			}
		}

		fmt.Println("✓ Shell integration installed")
		fmt.Printf("  Location: %s\n", rcFile)
		fmt.Printf("  Completion: %s\n", completionPath)
		fmt.Println("\nAvailable commands:")
		fmt.Println("  cw              - Interactive super-prompt (workspaces, clones, actions)")
		fmt.Println("  cw start <name> - Start a workspace")
		fmt.Println("  cw create       - Create a workspace")
		fmt.Println("\n✓ Tab completion enabled for all cw commands")
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
