package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/git"
	"github.com/pmossman/claude-workspace/internal/template"
	"github.com/pmossman/claude-workspace/internal/workspace"
	"github.com/spf13/cobra"
)

var (
	createSummary string
	createRemote  string
)

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new workspace (interactive)",
	Long: `Creates a new workspace with clone management.

Interactive mode (recommended):
  cw create

Direct mode:
  cw create feature-auth --remote airbyte

Legacy mode (without clone management):
  cw create feature-auth ~/dev/my-repo`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		var name string
		var absRepoPath string

		// Interactive mode if no args provided
		if len(args) == 0 && createRemote == "" {
			return interactiveCreate(cfg)
		}

		// Get name from args
		if len(args) > 0 {
			name = args[0]
		} else {
			return fmt.Errorf("workspace name required when using --remote")
		}

		// Determine mode: remote-based or path-based
		if createRemote != "" {
			// Remote-based mode: find or create clone
			absRepoPath, err = findOrCreateClone(cfg, name, createRemote)
			if err != nil {
				return err
			}
		} else if len(args) == 2 {
			// Legacy path-based mode
			repoPath := args[1]

			// Expand ~ in path
			if len(repoPath) >= 2 && repoPath[:2] == "~/" {
				home, _ := os.UserHomeDir()
				repoPath = filepath.Join(home, repoPath[2:])
			}

			// Make path absolute
			absRepoPath, err = filepath.Abs(repoPath)
			if err != nil {
				return fmt.Errorf("invalid repo path: %w", err)
			}

			// Check if repo path exists
			if _, err := os.Stat(absRepoPath); os.IsNotExist(err) {
				return fmt.Errorf("repo path does not exist: %s", absRepoPath)
			}
		} else {
			return fmt.Errorf("must specify either --remote or <repo-path>")
		}

		// Add workspace to config
		if err := cfg.AddWorkspace(name, absRepoPath); err != nil {
			return err
		}

		// Set ClonePath for new format
		ws, _ := cfg.GetWorkspace(name)
		ws.ClonePath = absRepoPath

		// If using remote-based mode, assign clone to workspace
		if createRemote != "" {
			if err := cfg.AssignCloneToWorkspace(absRepoPath, name); err != nil {
				return err
			}
		}

		// Create workspace directory structure
		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)
		if err := wsMgr.Create(name); err != nil {
			return err
		}

		// Write initial summary if provided
		if createSummary != "" {
			summaryPath := filepath.Join(wsMgr.GetPath(name), "summary.txt")
			if err := os.WriteFile(summaryPath, []byte(createSummary), 0644); err != nil {
				return fmt.Errorf("failed to write summary: %w", err)
			}
		}

		// Generate CLAUDE.md in repo
		workspaceDir := wsMgr.GetPath(name)
		if err := template.GenerateClaudeMd(name, workspaceDir, absRepoPath); err != nil {
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

		fmt.Printf("✓ Created workspace '%s'\n", name)
		fmt.Printf("  Repository: %s\n", absRepoPath)
		if createRemote != "" {
			fmt.Printf("  Remote: %s\n", createRemote)
		}
		fmt.Printf("  Workspace dir: %s\n", workspaceDir)
		fmt.Println("\nNext: claude-workspace start", name)

		return nil
	},
}

// findOrCreateClone finds a free clone or prompts user to create/takeover
func findOrCreateClone(cfg *config.Config, workspaceName, remoteName string) (string, error) {
	// Get remote (validates it exists)
	_, err := cfg.GetRemote(remoteName)
	if err != nil {
		return "", err
	}

	// Reopen /dev/tty for both reading and writing to ensure output is displayed immediately
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", fmt.Errorf("failed to open terminal: %w", err)
	}
	defer tty.Close()

	// Try to find a free clone
	freeClone := cfg.FindFreeClone(remoteName)

	// Check for idle clones
	idleClones := cfg.FindIdleClones(remoteName)

	// Build options list
	fmt.Fprintln(tty)
	if freeClone != nil {
		fmt.Fprintf(tty, "Found free clone: %s\n", freeClone.Path)
		fmt.Fprintln(tty)
		fmt.Fprintln(tty, "Options:")
		fmt.Fprintf(tty, "  1. Use free clone: %s\n", freeClone.Path)
		fmt.Fprintln(tty, "  2. Create a new clone")

		optionOffset := 3
		for i, clone := range idleClones {
			ws, _ := cfg.GetWorkspace(clone.InUseBy)
			fmt.Fprintf(tty, "  %d. Take over clone from '%s' (idle, branch: %s)\n", i+optionOffset, ws.Name, clone.CurrentBranch)
		}
	} else {
		fmt.Fprintf(tty, "No free clones available for '%s'\n", remoteName)
		fmt.Fprintln(tty)
		fmt.Fprintln(tty, "Options:")
		fmt.Fprintln(tty, "  1. Create a new clone")

		optionOffset := 2
		for i, clone := range idleClones {
			ws, _ := cfg.GetWorkspace(clone.InUseBy)
			fmt.Fprintf(tty, "  %d. Take over clone from '%s' (idle, branch: %s)\n", i+optionOffset, ws.Name, clone.CurrentBranch)
		}
	}

	fmt.Fprintln(tty, "  0. Cancel")
	fmt.Fprintln(tty)
	fmt.Fprint(tty, "Choice: ")

	reader := bufio.NewReader(tty)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Parse choice
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 0 {
		return "", fmt.Errorf("invalid choice")
	}

	if choice == 0 {
		return "", fmt.Errorf("cancelled")
	}

	// Handle choices based on whether we have a free clone
	if freeClone != nil {
		switch choice {
		case 1:
			// Use free clone
			return freeClone.Path, nil
		case 2:
			// Create new clone
			return createNewClone(cfg, remoteName)
		default:
			// Take over idle clone
			idx := choice - 3
			if idx >= 0 && idx < len(idleClones) {
				clone := idleClones[idx]
				oldWorkspace := clone.InUseBy
				if err := cfg.FreeClone(clone.Path); err != nil {
					return "", err
				}
				fmt.Fprintf(tty, "Took over clone from workspace '%s'\n", oldWorkspace)
				return clone.Path, nil
			}
			return "", fmt.Errorf("invalid choice")
		}
	} else {
		switch choice {
		case 1:
			// Create new clone
			return createNewClone(cfg, remoteName)
		default:
			// Take over idle clone
			idx := choice - 2
			if idx >= 0 && idx < len(idleClones) {
				clone := idleClones[idx]
				oldWorkspace := clone.InUseBy
				if err := cfg.FreeClone(clone.Path); err != nil {
					return "", err
				}
				fmt.Fprintf(tty, "Took over clone from workspace '%s'\n", oldWorkspace)
				return clone.Path, nil
			}
			return "", fmt.Errorf("invalid choice")
		}
	}
}

// createNewClone creates a new clone of a remote
func createNewClone(cfg *config.Config, remoteName string) (string, error) {
	remote, err := cfg.GetRemote(remoteName)
	if err != nil {
		return "", err
	}

	// Reopen /dev/tty for writing
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		// Fallback to stdout if tty not available
		tty = os.Stdout
	} else {
		defer tty.Close()
	}

	// Get next clone number
	cloneNum := cfg.GetNextCloneNumber(remoteName)
	clonePath := filepath.Join(remote.CloneBaseDir, fmt.Sprintf("%d", cloneNum))

	fmt.Fprintf(tty, "\nCreating clone %d...\n", cloneNum)
	fmt.Fprintf(tty, "  Cloning from: %s\n", remote.URL)
	fmt.Fprintf(tty, "  To: %s\n", clonePath)
	fmt.Fprintln(tty)

	// Clone the repository
	if err := git.Clone(remote.URL, clonePath); err != nil {
		return "", err
	}

	// Add clone to config
	if err := cfg.AddClone(clonePath, remoteName); err != nil {
		return "", err
	}

	// Get current branch
	branch, err := git.GetCurrentBranch(clonePath)
	if err != nil {
		branch = "unknown"
	}

	clone, _ := cfg.GetClone(clonePath)
	clone.CurrentBranch = branch

	fmt.Fprintf(tty, "✓ Created clone at %s\n\n", clonePath)
	return clonePath, nil
}

// interactiveCreate prompts user for workspace details
func interactiveCreate(cfg *config.Config) error {
	// Reopen /dev/tty for both reading and writing to ensure we can interact with terminal after fzf
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open terminal: %w", err)
	}
	defer tty.Close()

	reader := bufio.NewReader(tty)

	// Check if any remotes exist
	if len(cfg.Remotes) == 0 {
		fmt.Fprintln(tty, "No remotes registered yet.")
		fmt.Fprintln(tty, "\nFirst, add a remote:")
		fmt.Fprintln(tty, "  cw add-remote <name> <git-url> --clone-dir <path>")
		fmt.Fprintln(tty, "\nExample:")
		fmt.Fprintln(tty, "  cw add-remote airbyte git@github.com:airbytehq/airbyte-platform-internal.git --clone-dir ~/dev/airbyte-clones")
		return fmt.Errorf("no remotes available")
	}

	// Prompt for workspace name
	fmt.Fprintln(tty)
	fmt.Fprint(tty, "Workspace name: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}

	// Check if workspace already exists
	if _, err := cfg.GetWorkspace(name); err == nil {
		return fmt.Errorf("workspace '%s' already exists", name)
	}

	// Select remote
	var remoteNames []string
	for remoteName := range cfg.Remotes {
		remoteNames = append(remoteNames, remoteName)
	}
	sort.Strings(remoteNames)

	var remoteName string

	if len(remoteNames) == 1 {
		// Auto-select if only one remote
		remoteName = remoteNames[0]
		clones := cfg.GetClonesForRemote(remoteName)
		freeCount := 0
		for _, clone := range clones {
			if clone.InUseBy == "" {
				freeCount++
			}
		}
		fmt.Fprintln(tty)
		fmt.Fprintf(tty, "Using remote: %s (%d clones, %d free)\n", remoteName, len(clones), freeCount)
	} else {
		// Multiple remotes - show list and prompt
		fmt.Fprintln(tty)
		fmt.Fprintln(tty, "Available remotes:")
		for _, rName := range remoteNames {
			clones := cfg.GetClonesForRemote(rName)
			freeCount := 0
			for _, clone := range clones {
				if clone.InUseBy == "" {
					freeCount++
				}
			}
			fmt.Fprintf(tty, "  %s (%d clones, %d free)\n", rName, len(clones), freeCount)
		}
		fmt.Fprintln(tty)
		fmt.Fprint(tty, "Select remote: ")

		remoteName, _ = reader.ReadString('\n')
		remoteName = strings.TrimSpace(remoteName)

		// Validate remote exists
		if _, err := cfg.GetRemote(remoteName); err != nil {
			return fmt.Errorf("remote '%s' not found", remoteName)
		}
	}

	// Auto-generate summary from name
	autoSummary := generateSummary(name)
	fmt.Fprintln(tty)
	fmt.Fprintf(tty, "Auto-generated summary: %s\n", autoSummary)
	fmt.Fprint(tty, "Press Enter to accept, or type a custom summary: ")
	customSummary, _ := reader.ReadString('\n')
	customSummary = strings.TrimSpace(customSummary)

	summary := autoSummary
	if customSummary != "" {
		summary = customSummary
	}

	// Find or create clone
	absRepoPath, err := findOrCreateClone(cfg, name, remoteName)
	if err != nil {
		return err
	}

	// Create workspace
	if err := cfg.AddWorkspace(name, absRepoPath); err != nil {
		return err
	}

	ws, _ := cfg.GetWorkspace(name)
	ws.ClonePath = absRepoPath

	// Assign clone to workspace
	if err := cfg.AssignCloneToWorkspace(absRepoPath, name); err != nil {
		return err
	}

	// Create workspace directory structure
	wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)
	if err := wsMgr.Create(name); err != nil {
		return err
	}

	// Write summary
	summaryPath := filepath.Join(wsMgr.GetPath(name), "summary.txt")
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	// Generate CLAUDE.md in repo
	workspaceDir := wsMgr.GetPath(name)
	if err := template.GenerateClaudeMd(name, workspaceDir, absRepoPath); err != nil {
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

	fmt.Println()
	fmt.Printf("✓ Created workspace '%s'\n", name)
	fmt.Printf("  Repository: %s\n", absRepoPath)
	fmt.Printf("  Remote: %s\n", remoteName)
	fmt.Printf("  Summary: %s\n", summary)
	fmt.Printf("  Workspace dir: %s\n", workspaceDir)
	fmt.Println("\nNext: cw start", name)

	return nil
}

// generateSummary creates a human-readable summary from a workspace name
func generateSummary(name string) string {
	// Replace hyphens and underscores with spaces
	summary := strings.ReplaceAll(name, "-", " ")
	summary = strings.ReplaceAll(summary, "_", " ")

	// Capitalize first letter of each word
	words := strings.Fields(summary)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

func init() {
	createCmd.Flags().StringVar(&createSummary, "summary", "", "Initial workspace summary (optional, Claude will update it)")
	createCmd.Flags().StringVar(&createRemote, "remote", "", "Remote to use for clone management")
}
