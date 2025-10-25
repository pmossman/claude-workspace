package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/pmossman/claude-workspace/internal/config"
	"github.com/pmossman/claude-workspace/internal/session"
	"github.com/pmossman/claude-workspace/internal/workspace"
	"github.com/spf13/cobra"
)

var selectCmd = &cobra.Command{
	Use:   "select",
	Short: "Interactive super-prompt for all workspace operations",
	Long:  `Opens an interactive fzf menu to choose workspaces, create new ones, browse clones, etc. This is the default command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if fzf is installed
		if err := checkFzfInstalled(); err != nil {
			return err
		}

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)
		sessionMgr := session.NewManager()

		// ANSI color codes
		const (
			colorReset  = "\033[0m"
			colorGray   = "\033[90m"
			colorCyan   = "\033[36m"
			colorGreen  = "\033[32m"
			colorYellow = "\033[33m"
			colorBlue   = "\033[34m"
		)

		// Build menu options
		var inputLines []string

		// Add workspaces section header FIRST
		if len(cfg.Workspaces) > 0 {
			inputLines = append(inputLines, colorGray+"──── WORKSPACES ────"+colorReset)
		}

		// Build workspace list sorted by last active
		type wsEntry struct {
			name string
			ws   *config.Workspace
		}
		var entries []wsEntry
		for name, ws := range cfg.Workspaces {
			entries = append(entries, wsEntry{name: name, ws: ws})
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].ws.LastActive.After(entries[j].ws.LastActive)
		})

		// Add workspace items AFTER header
		for _, entry := range entries {
			ws := entry.ws
			summary := wsMgr.GetSummary(entry.name)
			lastActive := formatTimeAgo(ws.LastActive)

			// Get tmux session state
			sessionName := sessionMgr.GetSessionName(entry.name)
			sessionState, err := sessionMgr.GetSessionState(sessionName)
			if err != nil {
				sessionState = "unknown"
			}

			// Color code status based on session state
			statusColor := colorGray
			if sessionState == "attached" {
				statusColor = colorGreen
			} else if sessionState == "detached" {
				statusColor = colorYellow
			}

			// Format: name [status] summary (time)
			line := fmt.Sprintf("%s %s[%s]%s %s %s(%s)%s",
				colorCyan+entry.name+colorReset,
				statusColor,
				sessionState,
				colorReset,
				summary,
				colorGray,
				lastActive,
				colorReset,
			)
			inputLines = append(inputLines, line)
		}

		if len(cfg.Workspaces) > 0 {
			inputLines = append(inputLines, "")
		}

		// Add actions section header FIRST
		inputLines = append(inputLines, colorGray+"──── ACTIONS ────"+colorReset)

		// Add action items AFTER header
		inputLines = append(inputLines, colorBlue+"→"+colorReset+" Create new workspace")

		if len(cfg.Workspaces) > 0 {
			inputLines = append(inputLines, colorBlue+"→"+colorReset+" Archive workspace")
		}

		if len(cfg.Clones) > 0 {
			inputLines = append(inputLines, fmt.Sprintf(colorBlue+"→"+colorReset+" Browse clones "+colorGray+"(%d available)"+colorReset, len(cfg.Clones)))
		}

		if len(cfg.Remotes) > 0 {
			inputLines = append(inputLines, fmt.Sprintf(colorBlue+"→"+colorReset+" Create new clone "+colorGray+"(%d remotes)"+colorReset, len(cfg.Remotes)))
			inputLines = append(inputLines, fmt.Sprintf(colorBlue+"→"+colorReset+" List remotes "+colorGray+"(%d)"+colorReset, len(cfg.Remotes)))
		}

		input := strings.Join(inputLines, "\n")

		// Get path to self for preview command
		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		// Build fzf command with preview
		// Use sh -c to properly pass the selection as a quoted argument
		previewCmd := fmt.Sprintf("sh -c '%s preview-menu \"$1\"' _ {}", self)
		fzfCmd := exec.Command("fzf",
			"--ansi",
			"--no-sort",
			"--layout=reverse",
			"--height=100%",
			"--preview="+previewCmd,
			"--preview-window=right:50%:wrap",
			"--header=Select an option (Ctrl-C to cancel)",
			"--prompt=claude-workspace> ",
		)

		// Set up pipes
		fzfCmd.Stdin = strings.NewReader(input)
		fzfCmd.Stderr = os.Stderr

		var outBuf bytes.Buffer
		fzfCmd.Stdout = &outBuf

		// Run fzf
		if err := fzfCmd.Run(); err != nil {
			// User cancelled (Ctrl-C)
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() == 130 {
					return nil
				}
			}
			return fmt.Errorf("fzf failed: %w", err)
		}

		// Extract selection
		selected := strings.TrimSpace(outBuf.String())
		if selected == "" {
			return nil
		}

		// Strip ANSI color codes from selection
		selected = stripANSI(selected)

		// Handle actions
		if strings.HasPrefix(selected, "→") {
			return handleAction(cfg, selected)
		}

		// Handle section headers
		if strings.HasPrefix(selected, "────") {
			fmt.Println("Please select a workspace or action, not a section header")
			return nil
		}

		// Parse workspace name (everything before '[')
		bracketIdx := strings.Index(selected, "[")
		if bracketIdx == -1 {
			return fmt.Errorf("invalid selection format")
		}
		workspaceName := strings.TrimSpace(selected[:bracketIdx])

		// Call start command for the selected workspace
		return startCmd.RunE(cmd, []string{workspaceName})
	},
}

// handleAction handles the action items from the menu
func handleAction(cfg *config.Config, action string) error {
	switch {
	case strings.HasPrefix(action, "→ Create new workspace"):
		return createCmd.RunE(nil, []string{})

	case strings.HasPrefix(action, "→ Archive workspace"):
		return interactiveArchive(cfg)

	case strings.HasPrefix(action, "→ Browse clones"):
		return browseClones(cfg)

	case strings.HasPrefix(action, "→ Create new clone"):
		return interactiveNewClone(cfg)

	case strings.HasPrefix(action, "→ List remotes"):
		return listRemotesCmd.RunE(nil, []string{})

	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

// interactiveArchive shows an interactive workspace archive selector
func interactiveArchive(cfg *config.Config) error {
	if len(cfg.Workspaces) == 0 {
		fmt.Println("No workspaces to archive.")
		return nil
	}

	wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)

	// Build workspace list
	type wsEntry struct {
		name string
		ws   *config.Workspace
	}
	var entries []wsEntry
	for name, ws := range cfg.Workspaces {
		entries = append(entries, wsEntry{name: name, ws: ws})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ws.LastActive.After(entries[j].ws.LastActive)
	})

	var inputLines []string
	for _, entry := range entries {
		ws := entry.ws
		summary := wsMgr.GetSummary(entry.name)
		lastActive := formatTimeAgo(ws.LastActive)

		line := fmt.Sprintf("%s [%s] %s (%s)",
			entry.name,
			ws.Status,
			summary,
			lastActive,
		)
		inputLines = append(inputLines, line)
	}

	input := strings.Join(inputLines, "\n")

	fzfCmd := exec.Command("fzf",
		"--ansi",
		"--height=50%",
		"--header=Select workspace to archive (Ctrl-C to cancel)",
		"--prompt=Archive> ",
	)

	fzfCmd.Stdin = strings.NewReader(input)
	fzfCmd.Stderr = os.Stderr

	var outBuf bytes.Buffer
	fzfCmd.Stdout = &outBuf

	if err := fzfCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 {
				return nil
			}
		}
		return fmt.Errorf("fzf failed: %w", err)
	}

	selected := strings.TrimSpace(outBuf.String())
	if selected == "" {
		return nil
	}

	// Parse workspace name (everything before '[')
	bracketIdx := strings.Index(selected, "[")
	if bracketIdx == -1 {
		return fmt.Errorf("invalid selection format")
	}
	workspaceName := strings.TrimSpace(selected[:bracketIdx])

	// Call archive command
	return archiveCmd.RunE(nil, []string{workspaceName})
}

// browseClones shows an interactive clone browser
func browseClones(cfg *config.Config) error {
	if len(cfg.Clones) == 0 {
		fmt.Println("No clones available.")
		fmt.Println("Create one with: cw new-clone <remote>")
		return nil
	}

	// Build clone list
	var inputLines []string
	for _, clone := range cfg.Clones {
		status := "free"
		if clone.InUseBy != "" {
			status = fmt.Sprintf("in use by: %s", clone.InUseBy)
		}
		line := fmt.Sprintf("%s [%s] %s", clone.Path, clone.RemoteName, status)
		inputLines = append(inputLines, line)
	}

	input := strings.Join(inputLines, "\n")

	fzfCmd := exec.Command("fzf",
		"--ansi",
		"--height=100%",
		"--header=Clone paths (use 'cwc' to cd interactively, or copy path below)",
		"--prompt=Clone> ",
	)

	fzfCmd.Stdin = strings.NewReader(input)
	fzfCmd.Stderr = os.Stderr

	var outBuf bytes.Buffer
	fzfCmd.Stdout = &outBuf

	if err := fzfCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 {
				return nil
			}
		}
		return fmt.Errorf("fzf failed: %w", err)
	}

	selected := strings.TrimSpace(outBuf.String())
	if selected == "" {
		return nil
	}

	// Extract clone path (everything before '[')
	bracketIdx := strings.Index(selected, "[")
	if bracketIdx == -1 {
		return nil
	}
	clonePath := strings.TrimSpace(selected[:bracketIdx])

	// Output CD marker for shell function to detect
	fmt.Printf("CD:%s\n", clonePath)

	return nil
}

// interactiveNewClone prompts for remote and creates a new clone
func interactiveNewClone(cfg *config.Config) error {
	if len(cfg.Remotes) == 0 {
		fmt.Println("No remotes registered.")
		fmt.Println("Add one with: cw add-remote <name> <url> --clone-dir <path>")
		return nil
	}

	// Build remote list
	var remoteNames []string
	for name := range cfg.Remotes {
		remoteNames = append(remoteNames, name)
	}
	sort.Strings(remoteNames)

	var inputLines []string
	for _, name := range remoteNames {
		remote := cfg.Remotes[name]
		cloneCount := len(cfg.GetClonesForRemote(name))
		line := fmt.Sprintf("%s (%d clones) - %s", name, cloneCount, remote.URL)
		inputLines = append(inputLines, line)
	}

	input := strings.Join(inputLines, "\n")

	fzfCmd := exec.Command("fzf",
		"--ansi",
		"--height=50%",
		"--header=Select remote to clone",
		"--prompt=Remote> ",
	)

	fzfCmd.Stdin = strings.NewReader(input)
	fzfCmd.Stderr = os.Stderr

	var outBuf bytes.Buffer
	fzfCmd.Stdout = &outBuf

	if err := fzfCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 {
				return nil
			}
		}
		return fmt.Errorf("fzf failed: %w", err)
	}

	selected := strings.TrimSpace(outBuf.String())
	if selected == "" {
		return nil
	}

	// Extract remote name (before first space)
	parts := strings.Fields(selected)
	if len(parts) == 0 {
		return fmt.Errorf("invalid selection")
	}
	remoteName := parts[0]

	// Call new-clone command
	return newCloneCmd.RunE(nil, []string{remoteName})
}

// stripANSI removes ANSI color codes from a string
func stripANSI(s string) string {
	// Use a more sophisticated approach that handles UTF-8 properly
	// Match ANSI escape sequences: \033[ followed by any chars until 'm'
	result := strings.Builder{}
	i := 0
	for i < len(s) {
		// Check for ANSI escape sequence start
		if i < len(s)-1 && (s[i] == '\033' || s[i] == '\x1b') && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find 'm'
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++ // skip the 'm'
			}
		} else {
			// Regular character - write it as-is (preserves UTF-8)
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// previewMenuCmd handles previews for the super-prompt menu
var previewMenuCmd = &cobra.Command{
	Use:    "preview-menu <selection>",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		selection := strings.Join(args, " ")

		// Strip ANSI color codes from selection
		selection = stripANSI(selection)

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Handle different selection types
		if strings.HasPrefix(selection, "→ Create new workspace") {
			fmt.Println("Create a new workspace with a fresh clone or existing repo.")
			fmt.Println()
			fmt.Println("This will:")
			fmt.Println("  • Prompt for workspace name")
			fmt.Println("  • Let you choose a remote")
			fmt.Println("  • Auto-find or create a clone")
			fmt.Println("  • Set up workspace tracking files")
			return nil
		}

		if strings.HasPrefix(selection, "→ Archive workspace") {
			fmt.Println("Archive an existing workspace.")
			fmt.Println()
			fmt.Printf("Total workspaces: %d\n", len(cfg.Workspaces))
			fmt.Println()
			fmt.Println("This will:")
			fmt.Println("  • Select a workspace to archive")
			fmt.Println("  • Move it to archived/ directory")
			fmt.Println("  • Free up the clone if managed")
			fmt.Println("  • Preserve all workspace files")
			return nil
		}

		if strings.HasPrefix(selection, "→ Browse clones") {
			fmt.Println("Browse all available clones.")
			fmt.Println()
			fmt.Printf("Total clones: %d\n", len(cfg.Clones))
			freeCount := 0
			for _, clone := range cfg.Clones {
				if clone.InUseBy == "" {
					freeCount++
				}
			}
			fmt.Printf("Free clones: %d\n", freeCount)
			fmt.Printf("In use: %d\n", len(cfg.Clones)-freeCount)
			fmt.Println()
			fmt.Println("Select a clone to cd into it.")
			return nil
		}

		if strings.HasPrefix(selection, "→ Create new clone") {
			fmt.Println("Create a new numbered clone from a remote.")
			fmt.Println()
			fmt.Printf("Available remotes: %d\n", len(cfg.Remotes))
			fmt.Println()
			fmt.Println("This will:")
			fmt.Println("  • Prompt to select a remote")
			fmt.Println("  • Clone to next available number")
			fmt.Println("  • Track the clone for future use")
			return nil
		}

		if strings.HasPrefix(selection, "→ List remotes") {
			fmt.Println("View all registered remotes.")
			fmt.Println()
			fmt.Printf("Total remotes: %d\n", len(cfg.Remotes))
			fmt.Println()
			fmt.Println("Shows:")
			fmt.Println("  • Remote name")
			fmt.Println("  • Git URL")
			fmt.Println("  • Clone base directory")
			fmt.Println("  • Number of clones")
			return nil
		}

		if strings.HasPrefix(selection, "────") {
			// Section header - no preview
			return nil
		}

		// Must be a workspace - show workspace preview
		bracketIdx := strings.Index(selection, "[")
		if bracketIdx == -1 {
			// Not a workspace, no preview
			return nil
		}
		workspaceName := strings.TrimSpace(selection[:bracketIdx])

		return showWorkspacePreview(cfg, workspaceName)
	},
}

// showWorkspacePreview shows detailed workspace information
func showWorkspacePreview(cfg *config.Config, name string) error {
	ws, err := cfg.GetWorkspace(name)
	if err != nil {
		return err
	}

	wsMgr := workspace.NewManager(cfg.Settings.WorkspaceDir)

	fmt.Printf("WORKSPACE: %s\n", name)
	fmt.Printf("STATUS: %s", formatStatus(ws.Status))
	if ws.Status == config.StatusActive {
		fmt.Printf(" (PID %d)", ws.SessionPID)
	}
	fmt.Println()
	fmt.Printf("REPO: %s\n", ws.GetRepoPath())

	// Show clone info if managed
	if ws.ClonePath != "" {
		if clone, err := cfg.GetClone(ws.ClonePath); err == nil {
			fmt.Printf("REMOTE: %s\n", clone.RemoteName)
			fmt.Printf("BRANCH: %s\n", clone.CurrentBranch)
		}
	}

	fmt.Printf("LAST ACTIVE: %s\n", formatTimeAgo(ws.LastActive))

	summary := wsMgr.GetSummary(name)
	if summary != "(no summary)" {
		fmt.Printf("SUMMARY: %s\n", summary)
	}

	// Show continuation
	continuation := wsMgr.GetContinuation(name)
	if continuation != "" {
		fmt.Println()
		fmt.Println("─── CONTINUATION ───")
		// Truncate if too long
		if len(continuation) > 500 {
			fmt.Println(continuation[:500] + "...")
		} else {
			fmt.Println(continuation)
		}
	}

	// Show context preview
	context := wsMgr.GetContext(name)
	if context != "(no context yet)" {
		fmt.Println()
		fmt.Println("─── RECENT CONTEXT ───")
		fmt.Println(context)
	}

	return nil
}

// preview is a hidden command used by fzf to generate previews (for cw start)
var previewCmd = &cobra.Command{
	Use:    "preview <name>",
	Hidden: true,
	Args:   cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Join all args to handle workspace names with spaces
		name := strings.Join(args, " ")

		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return showWorkspacePreview(cfg, name)
	},
}

func init() {
	rootCmd.AddCommand(previewCmd)
	rootCmd.AddCommand(previewMenuCmd)
}

func checkFzfInstalled() error {
	cmd := exec.Command("fzf", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fzf is not installed. Please install fzf to use the interactive selector.\n" +
			"Install with: brew install fzf (macOS) or see https://github.com/junegunn/fzf")
	}
	return nil
}
