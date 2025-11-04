package cmd

import (
	"fmt"
	"sort"

	"github.com/pmossman/claudew/internal/config"
	"github.com/spf13/cobra"
)

var listRemotesCmd = &cobra.Command{
	Use:   "list-remotes",
	Short: "List all registered remotes",
	Long:  `Lists all registered remotes with their URLs and clone directories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if len(cfg.Remotes) == 0 {
			fmt.Println("No remotes registered.")
			fmt.Println("\nAdd a remote with: claudew add-remote <name> <git-url> --clone-dir <path>")
			return nil
		}

		// Sort remotes by name
		var names []string
		for name := range cfg.Remotes {
			names = append(names, name)
		}
		sort.Strings(names)

		// Print header
		fmt.Printf("%-15s %-50s %s\n", "NAME", "URL", "CLONE DIRECTORY")
		fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────────────────────")

		// Print remotes
		for _, name := range names {
			remote := cfg.Remotes[name]

			// Count clones
			clones := cfg.GetClonesForRemote(name)
			freeCount := 0
			for _, clone := range clones {
				if clone.InUseBy == "" {
					freeCount++
				}
			}

			url := remote.URL
			if len(url) > 50 {
				url = url[:47] + "..."
			}

			fmt.Printf("%-15s %-50s %s\n", name, url, remote.CloneBaseDir)
			if len(clones) > 0 {
				fmt.Printf("  └─ %d clones (%d free, %d in use)\n", len(clones), freeCount, len(clones)-freeCount)
			} else {
				fmt.Printf("  └─ No clones yet\n")
			}
		}

		return nil
	},
}
