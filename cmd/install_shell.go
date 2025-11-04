package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Embed shell integration files from the cmd/shell/ directory
// These are copied to ~/.claudew/ during installation
var (
	//go:embed shell/shell-integration.sh
	shellIntegrationScript string

	//go:embed shell/completion.zsh
	zshCompletionSetup string

	//go:embed shell/completion.bash
	bashCompletionSetup string
)

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

		// Create ~/.claudew directory for integration files
		claudewDir := filepath.Join(home, ".claudew")
		if err := os.MkdirAll(claudewDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", claudewDir, err)
		}

		// Write shell integration to ~/.claudew/shell-integration.sh
		shellIntegrationPath := filepath.Join(claudewDir, "shell-integration.sh")
		if err := os.WriteFile(shellIntegrationPath, []byte(shellIntegrationScript), 0644); err != nil {
			return fmt.Errorf("failed to write shell integration: %w", err)
		}

		// Generate and write completion files
		var completionScript string
		var completionPath string
		var completionSetupPath string
		var completionSetupContent string

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

			// Remove the extra "compdef _claudew claudew" line that Cobra adds (line 2)
			lines := strings.Split(completionScript, "\n")
			if len(lines) > 1 && strings.HasPrefix(lines[1], "compdef ") {
				completionScript = strings.Join(append(lines[:1], lines[2:]...), "\n")
			}

			// Write completion setup to ~/.claudew/completion.zsh
			completionSetupPath = filepath.Join(claudewDir, "completion.zsh")
			completionSetupContent = zshCompletionSetup
		} else {
			// Generate bash completion
			completionPath = filepath.Join(home, ".claudew-completion.bash")

			// Generate completion script
			var builder strings.Builder
			if err := rootCmd.GenBashCompletion(&builder); err != nil {
				return fmt.Errorf("failed to generate bash completion: %w", err)
			}
			completionScript = builder.String()

			// Write completion setup to ~/.claudew/completion.bash
			completionSetupPath = filepath.Join(claudewDir, "completion.bash")
			completionSetupContent = bashCompletionSetup
		}

		// Write completion script
		if err := os.WriteFile(completionPath, []byte(completionScript), 0644); err != nil {
			return fmt.Errorf("failed to write completion script: %w", err)
		}

		// Write completion setup file
		if err := os.WriteFile(completionSetupPath, []byte(completionSetupContent), 0644); err != nil {
			return fmt.Errorf("failed to write completion setup: %w", err)
		}

		// If force installing and already installed, remove old sections first
		if installShellForce && installed {
			content, err := os.ReadFile(rcFile)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", rcFile, err)
			}

			// Remove existing claudew sections
			newContent := removeClaudewSections(string(content))

			// Write back the cleaned content
			if err := os.WriteFile(rcFile, []byte(newContent), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", rcFile, err)
			}
		}

		// Append source statements to rc file
		f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", rcFile, err)
		}
		defer f.Close()

		rcAdditions := fmt.Sprintf(`
# claudew shell integration - managed by 'claudew install-shell'
[ -f %s ] && source %s
[ -f %s ] && source %s
`, shellIntegrationPath, shellIntegrationPath, completionSetupPath, completionSetupPath)

		if _, err := f.WriteString(rcAdditions); err != nil {
			return fmt.Errorf("failed to write to %s: %w", rcFile, err)
		}

		fmt.Println("✓ Shell integration installed")
		fmt.Printf("  Shell config: %s\n", rcFile)
		fmt.Printf("  Integration: %s\n", shellIntegrationPath)
		fmt.Printf("  Completion: %s\n", completionPath)
		fmt.Println("\nAvailable commands:")
		fmt.Println("  claudew              - Interactive super-prompt (workspaces, clones, actions)")
		fmt.Println("  claudew start <name> - Start a workspace")
		fmt.Println("  claudew create       - Create a workspace")
		fmt.Println("\n✓ Tab completion enabled")
		fmt.Println("\nNote: The 'cw' alias is automatically created for shorter typing")
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

		// Clean up ~/.claudew directory
		claudewDir := filepath.Join(home, ".claudew")
		os.RemoveAll(claudewDir) // Remove directory and all contents

		fmt.Println("✓ Shell integration uninstalled")
		fmt.Printf("  Cleaned up: %s\n", rcFile)
		fmt.Println("\nTo reinstall:")
		fmt.Println("  claudew install-shell")
		fmt.Println("\nReload your shell:")
		fmt.Printf("  source %s\n", rcFile)

		return nil
	},
}

// removeClaudewSections removes all claudew shell integration sections from rc file content
func removeClaudewSections(content string) string {
	markers := []string{
		"# claude-workspace shell integration",
		"# claudew shell integration",
	}

	lines := strings.Split(content, "\n")
	var newLines []string
	skipUntilBlank := false

	for i, line := range lines {
		// Check if this line is a marker
		isMarker := false
		for _, marker := range markers {
			if strings.Contains(line, marker) {
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
			// Skip until we hit a blank line
			trimmed := strings.TrimSpace(line)

			if trimmed == "" {
				// Found blank line - check if next line is also integration-related
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					// If next line is a known integration marker, keep skipping
					isNextMarker := false
					for _, marker := range markers {
						if strings.Contains(nextLine, marker) {
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

	return strings.Join(newLines, "\n")
}

func init() {
	installShellCmd.Flags().BoolVarP(&installShellForce, "force", "f", false, "Force reinstall even if already installed")
}
