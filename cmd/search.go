package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/claude"
)

var (
	searchScope string
	searchCwd   string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search Claude conversation history",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runSearch,
}

func init() {
	searchCmd.Flags().StringVar(&searchScope, "scope", "c", "search scope: c=conversation, f=files, t=tasks")
	searchCmd.Flags().StringVar(&searchCwd, "cwd", "", "limit search to specific working directory")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	// If cwd specified, search just that project
	if searchCwd != "" {
		return searchProject(query, searchCwd)
	}

	// Otherwise, search the latest session across all known projects
	// For MVP, require --cwd
	return fmt.Errorf("please specify --cwd for search (global search coming in Phase 3)")
}

func searchProject(query, cwd string) error {
	jsonlPath, err := claude.LatestSessionFile(cfg.ClaudeProjectDir, cwd)
	if err != nil {
		return fmt.Errorf("no Claude session found for %s", cwd)
	}

	results, err := claude.SearchConversation(jsonlPath, query)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("No matches found.")
		return nil
	}

	for _, r := range results {
		prefix := "  "
		if r.Type == "user" {
			prefix = "> "
		}
		fmt.Printf("%s%s\n", prefix, r.Context)
	}
	return nil
}
