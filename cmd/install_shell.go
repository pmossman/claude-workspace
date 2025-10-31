package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const shellIntegration = `
# claudew shell integration
claudew() {
  # Pass through completion requests directly without capturing output
  if [ "$1" = "__complete" ]; then
    command claudew "$@"
    return $?
  fi

  # Only capture output for commands that may use CD: marker
  # All other commands pass through directly for real-time output
  case "$1" in
    cd|clones|select|"")
      # These commands might output CD: marker for navigation
      local output
      output=$(command claudew "$@" 2>&1)
      local exit_code=$?

      # Check if output contains CD: marker (for clone navigation)
      # Use CD::: as delimiter to handle paths with colons
      if echo "$output" | grep -q "^CD:::"; then
        local clone_path=$(echo "$output" | grep "^CD:::" | sed 's/^CD::://')
        if [ -n "$clone_path" ]; then
          if [ -d "$clone_path" ]; then
            cd "$clone_path" || {
              echo "❌ Error: Failed to change directory to: $clone_path" >&2
              return 1
            }
            echo "📂 Changed to: $clone_path"
            return 0
          else
            echo "❌ Error: Directory does not exist: $clone_path" >&2
            return 1
          fi
        fi
      fi

      # Otherwise, just display the output normally
      echo "$output"
      return $exit_code
      ;;
    *)
      # All other commands: pass through directly (no output buffering)
      command claudew "$@"
      return $?
      ;;
  esac
}
`

// isShellIntegrationInstalled checks if shell integration is already installed
func isShellIntegrationInstalled() (bool, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Detect shell
	shell := os.Getenv("SHELL")
	var rcFile string
	if strings.Contains(shell, "zsh") {
		rcFile = filepath.Join(home, ".zshrc")
	} else if strings.Contains(shell, "bash") {
		rcFile = filepath.Join(home, ".bashrc")
	} else {
		return false, "", fmt.Errorf("unsupported shell: %s (only bash and zsh supported)", shell)
	}

	// Check if already installed
	content, err := os.ReadFile(rcFile)
	if err != nil && !os.IsNotExist(err) {
		return false, "", fmt.Errorf("failed to read %s: %w", rcFile, err)
	}

	// Check for either old or new shell integration markers
	hasOld := strings.Contains(string(content), "# claude-workspace shell integration")
	hasNew := strings.Contains(string(content), "# claudew shell integration")
	installed := hasOld || hasNew
	return installed, rcFile, nil
}

var (
	installShellForce bool
)

var installShellCmd = &cobra.Command{
	Use:   "install-shell",
	Short: "Install shell integration (adds claudew function to your shell)",
	Long: `Installs shell integration for interactive features.

Adds the 'claudew' function to your ~/.zshrc or ~/.bashrc which wraps the
binary and adds directory navigation capability.

  claudew - Interactive super-prompt with workspace management and clone navigation

You can create a short alias in your shell config if desired:
  alias cw='claudew'

Use --force to reinstall if already installed (useful after updates).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already installed
		installed, rcFile, err := isShellIntegrationInstalled()
		if err != nil {
			return err
		}

		if installed && !installShellForce {
			fmt.Println("✓ Shell integration already installed")
			fmt.Printf("  Location: %s\n", rcFile)
			fmt.Println("\nAvailable commands:")
			fmt.Println("  claudew              - Interactive super-prompt (workspaces, clones, actions)")
			fmt.Println("  claudew start <name> - Start a workspace")
			fmt.Println("  claudew create       - Create a workspace")
			fmt.Println("\nTip: Create an alias for shorter typing:")
			fmt.Println("  alias cw='claudew'")
			fmt.Println("\nTo reinstall or update: claudew install-shell --force")
			return nil
		}

		if installShellForce && installed {
			fmt.Println("⚠️  Force reinstalling shell integration...")
			fmt.Println()
		}

		home, _ := os.UserHomeDir()
		shell := os.Getenv("SHELL")

		// Generate and install completion
		var completionScript string
		var completionPath string
		if strings.Contains(shell, "zsh") {
			// Generate zsh completion
			completionDir := filepath.Join(home, ".zsh", "completion")
			if err := os.MkdirAll(completionDir, 0755); err != nil {
				return fmt.Errorf("failed to create completion directory: %w", err)
			}
			completionPath = filepath.Join(completionDir, "_claudew")

			// Generate completion script to string
			var builder strings.Builder
			if err := rootCmd.GenZshCompletion(&builder); err != nil {
				return fmt.Errorf("failed to generate zsh completion: %w", err)
			}
			completionScript = builder.String()
		} else {
			// Generate bash completion
			completionPath = filepath.Join(home, ".claudew-completion.bash")

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
			completionSetup := `
# claudew completion
fpath=(~/.zsh/completion $fpath)
if ! command -v compinit > /dev/null 2>&1; then
  autoload -Uz compinit && compinit
fi
compdef _claudew claudew
`
			if _, err := f.WriteString(completionSetup); err != nil {
				return fmt.Errorf("failed to write completion setup: %w", err)
			}
		} else {
			completionSetup := fmt.Sprintf(`
# claudew completion
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
		fmt.Println("  claudew              - Interactive super-prompt (workspaces, clones, actions)")
		fmt.Println("  claudew start <name> - Start a workspace")
		fmt.Println("  claudew create       - Create a workspace")
		fmt.Println("\n✓ Tab completion enabled for all claudew commands")
		fmt.Println("\nTip: Create an alias for shorter typing:")
		fmt.Println("  echo \"alias cw='claudew'\" >> " + rcFile)
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

var uninstallShellCmd = &cobra.Command{
	Use:   "uninstall-shell",
	Short: "Uninstall shell integration",
	Long: `Removes the shell integration from your ~/.zshrc or ~/.bashrc.

This will remove:
- The claudew() shell function
- Completion setup
- Old claude-workspace integration (if present)

After uninstalling, you can reinstall with: claudew install-shell`,
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

		// Read current rc file
		content, err := os.ReadFile(rcFile)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rcFile, err)
		}

		originalContent := string(content)
		hasOldIntegration := strings.Contains(originalContent, "# claude-workspace shell integration")
		hasNewIntegration := strings.Contains(originalContent, "# claudew shell integration")

		if !hasOldIntegration && !hasNewIntegration {
			fmt.Println("✓ No shell integration found - nothing to uninstall")
			return nil
		}

		// Remove old integration markers
		markers := []string{
			"# claude-workspace shell integration",
			"# claudew shell integration",
			"# claude-workspace completion",
			"# claudew completion",
		}

		lines := strings.Split(originalContent, "\n")
		var newLines []string
		skipUntilBlank := false

		for i, line := range lines {
			// Check if this line is a marker
			isMarker := false
			for _, marker := range markers {
				if strings.TrimSpace(line) == marker {
					isMarker = true
					skipUntilBlank = true
					break
				}
			}

			if isMarker {
				// Skip this line and start looking for the end of the section
				continue
			}

			if skipUntilBlank {
				// Skip until we hit a blank line or a non-integration line
				trimmed := strings.TrimSpace(line)

				// Check if we've reached the end of the integration section
				// Integration ends at: blank line, or a line that starts with # but isn't part of completion
				if trimmed == "" {
					// Found blank line - check if next line is also integration-related
					if i+1 < len(lines) {
						nextLine := strings.TrimSpace(lines[i+1])
						// If next line is a known integration marker, keep skipping
						isNextMarker := false
						for _, marker := range markers {
							if nextLine == marker {
								isNextMarker = true
								break
							}
						}
						if isNextMarker {
							continue // Keep skipping
						}
					}
					skipUntilBlank = false
					newLines = append(newLines, line) // Keep the blank line
				}
				// Skip lines that look like integration content
				continue
			}

			// Keep this line
			newLines = append(newLines, line)
		}

		// Write back
		newContent := strings.Join(newLines, "\n")
		if err := os.WriteFile(rcFile, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", rcFile, err)
		}

		// Clean up completion files
		if strings.Contains(shell, "zsh") {
			oldCompPath := filepath.Join(home, ".zsh", "completion", "_claude-workspace")
			newCompPath := filepath.Join(home, ".zsh", "completion", "_claudew")
			os.Remove(oldCompPath) // Ignore errors
			os.Remove(newCompPath) // Ignore errors
		} else {
			oldCompPath := filepath.Join(home, ".claude-workspace-completion.bash")
			newCompPath := filepath.Join(home, ".claudew-completion.bash")
			os.Remove(oldCompPath) // Ignore errors
			os.Remove(newCompPath) // Ignore errors
		}

		fmt.Println("✓ Shell integration uninstalled")
		fmt.Printf("  Cleaned up: %s\n", rcFile)
		fmt.Println("\nTo reinstall:")
		fmt.Println("  claudew install-shell")
		fmt.Println("\nReload your shell:")
		fmt.Printf("  source %s\n", rcFile)

		return nil
	},
}

func init() {
	installShellCmd.Flags().BoolVarP(&installShellForce, "force", "f", false, "Force reinstall even if already installed")
}
