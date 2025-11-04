package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/pmossman/claudew/internal/config"
	"github.com/pmossman/claudew/internal/git"
	"github.com/spf13/cobra"
)

var (
	clonesInteractive bool
)

var clonesCmd = &cobra.Command{
	Use:   "clones [remote-name]",
	Short: "List all clones or clones for a specific remote",
	Long:  `Shows all clones with their paths, branches, and usage status.
Use -i/--interactive for fzf selection to cd into a clone.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.Clones) == 0 {
			fmt.Println("No clones registered.")
			fmt.Println("\nCreate a clone with: claudew new-clone <remote-name>")
			fmt.Println("Or import existing: claudew import-clone <remote-name> <path>")
			return nil
		}

		// Filter by remote if specified
		var remoteName string
		if len(args) > 0 {
			remoteName = args[0]
			if _, err := cfg.GetRemote(remoteName); err != nil {
				return err
			}
		}

		// Interactive mode
		if clonesInteractive {
			return interactiveCloneSelect(cfg, remoteName)
		}

		// Collect and sort clones
		type cloneEntry struct {
			path  string
			clone *config.Clone
		}
		var entries []cloneEntry

		for path, clone := range cfg.Clones {
			if remoteName == "" || clone.RemoteName == remoteName {
				entries = append(entries, cloneEntry{path: path, clone: clone})
			}
		}

		if len(entries) == 0 {
			fmt.Printf("No clones found for remote '%s'\n", remoteName)
			return nil
		}

		// Sort by remote name, then path
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].clone.RemoteName != entries[j].clone.RemoteName {
				return entries[i].clone.RemoteName < entries[j].clone.RemoteName
			}
			return entries[i].path < entries[j].path
		})

		// Print header
		fmt.Printf("%-40s %-12s %-15s %-10s %s\n", "CLONE PATH", "REMOTE", "BRANCH", "STATUS", "WORKSPACE")
		fmt.Println("──────────────────────────────────────────────────────────────────────────────────────────────────────────────")

		// Print clones
		currentRemote := ""
		for _, entry := range entries {
			clone := entry.clone

			// Print remote separator
			if clone.RemoteName != currentRemote {
				if currentRemote != "" {
					fmt.Println()
				}
				currentRemote = clone.RemoteName
			}

			// Update branch info
			branch, err := git.GetCurrentBranch(clone.Path)
			if err == nil && branch != clone.CurrentBranch {
				clone.CurrentBranch = branch
				cfg.Save() // Save updated branch
			}

			// Format status
			status := "free"
			workspace := "-"
			if clone.InUseBy != "" {
				ws, err := cfg.GetWorkspace(clone.InUseBy)
				if err == nil {
					status = string(ws.Status)
					workspace = clone.InUseBy
				} else {
					status = "orphaned"
					workspace = clone.InUseBy + " (missing)"
				}
			}

			// Truncate path if too long
			displayPath := clone.Path
			if len(displayPath) > 40 {
				displayPath = "..." + displayPath[len(displayPath)-37:]
			}

			fmt.Printf("%-40s %-12s %-15s %-10s %s\n",
				displayPath,
				clone.RemoteName,
				clone.CurrentBranch,
				status,
				workspace,
			)
		}

		return nil
	},
}

func interactiveCloneSelect(cfg *config.Config, remoteName string) error {
	// Check if fzf is installed
	if err := checkFzfInstalled(); err != nil {
		return err
	}

	// Collect clones
	type cloneEntry struct {
		path  string
		clone *config.Clone
	}
	var entries []cloneEntry

	for path, clone := range cfg.Clones {
		if remoteName == "" || clone.RemoteName == remoteName {
			entries = append(entries, cloneEntry{path: path, clone: clone})
		}
	}

	if len(entries) == 0 {
		if remoteName != "" {
			return fmt.Errorf("no clones found for remote '%s'", remoteName)
		}
		return fmt.Errorf("no clones found")
	}

	// Sort by remote name, then path
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].clone.RemoteName != entries[j].clone.RemoteName {
			return entries[i].clone.RemoteName < entries[j].clone.RemoteName
		}
		return entries[i].path < entries[j].path
	})

	// Build fzf input
	var inputLines []string
	for _, entry := range entries {
		clone := entry.clone

		// Update branch info
		branch, err := git.GetCurrentBranch(clone.Path)
		if err == nil && branch != clone.CurrentBranch {
			clone.CurrentBranch = branch
			cfg.Save()
		}

		// Format status
		status := "free"
		if clone.InUseBy != "" {
			ws, err := cfg.GetWorkspace(clone.InUseBy)
			if err == nil {
				status = fmt.Sprintf("%s:%s", string(ws.Status), clone.InUseBy)
			}
		}

		line := fmt.Sprintf("%s [%s] %s (%s)",
			clone.Path,
			clone.RemoteName,
			clone.CurrentBranch,
			status,
		)
		inputLines = append(inputLines, line)
	}

	input := strings.Join(inputLines, "\n")

	// Run fzf
	fzfCmd := exec.Command("fzf",
		"--ansi",
		"--no-sort",
		"--height=50%",
		"--header=Select a clone (Ctrl-C to cancel)",
		"--prompt=Clone> ",
	)

	fzfCmd.Stdin = strings.NewReader(input)
	fzfCmd.Stderr = os.Stderr

	var outBuf bytes.Buffer
	fzfCmd.Stdout = &outBuf

	if err := fzfCmd.Run(); err != nil {
		// User cancelled
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 130 {
				return nil
			}
		}
		return fmt.Errorf("fzf failed: %w", err)
	}

	// Extract selected path (first field)
	selected := strings.TrimSpace(outBuf.String())
	if selected == "" {
		return nil
	}

	parts := strings.Fields(selected)
	if len(parts) == 0 {
		return fmt.Errorf("invalid selection")
	}
	selectedPath := parts[0]

	// Output CD marker for shell function to detect
	// Use CD::: delimiter to handle paths with colons
	fmt.Printf("CD:::%s\n", selectedPath)

	return nil
}

func init() {
	clonesCmd.Flags().BoolVarP(&clonesInteractive, "interactive", "i", false, "Interactive clone selection with fzf")
}
