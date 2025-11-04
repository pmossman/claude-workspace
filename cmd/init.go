package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pmossman/claudew/internal/config"
	"github.com/pmossman/claudew/internal/session"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize claudew configuration",
	Long:  `Creates the configuration directory and initial config file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if tmux is installed
		sessionMgr := session.NewManager()
		if err := sessionMgr.CheckTmuxInstalled(); err != nil {
			return err
		}

		// Load or create default config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Save config to ensure directory structure exists
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println("✓ Initialized claudew")
		fmt.Printf("  Config directory: %s\n", cfg.Settings.WorkspaceDir)

		// Check if shell integration is already installed
		installed, _, err := isShellIntegrationInstalled()
		if err != nil {
			// If we can't check (unsupported shell), just show next steps
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Install shell integration: claudew install-shell")
			fmt.Println("  2. Add a remote: claudew add-remote <name> <git-url> --clone-dir <path>")
			fmt.Println("  3. Create a workspace: claudew create")
			fmt.Println("\nOr use the interactive selector: cw")
			return nil
		}

		if !installed {
			// Prompt to install shell integration
			fmt.Println()
			fmt.Print("Install shell integration now? [Y/n]: ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response == "" || response == "y" || response == "yes" {
				// Run install-shell command
				if err := installShellCmd.RunE(cmd, args); err != nil {
					return fmt.Errorf("failed to install shell integration: %w", err)
				}
			} else {
				fmt.Println("\nSkipped shell integration. You can install it later with:")
				fmt.Println("  claudew install-shell")
			}
		} else {
			fmt.Println("\n✓ Shell integration already installed")
		}

		fmt.Println("\nNext steps:")
		fmt.Println("  1. Add a remote: claudew add-remote <name> <git-url> --clone-dir <path>")
		fmt.Println("  2. Create a workspace: claudew create")
		fmt.Println("\nOr use the interactive selector: cw")

		return nil
	},
}
