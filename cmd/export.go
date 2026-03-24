package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/claude"
)

var exportOutput string

var exportCmd = &cobra.Command{
	Use:   "export <cwd>",
	Short: "Export Claude session as markdown",
	Args:  cobra.ExactArgs(1),
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file (default: stdout)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	cwd := args[0]

	jsonlPath, err := claude.LatestSessionFile(cfg.ClaudeProjectDir, cwd)
	if err != nil {
		return fmt.Errorf("no Claude session found for %s", cwd)
	}

	messages, err := claude.ParseJSONL(jsonlPath)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# Claude Session Export\n\n")
	b.WriteString(fmt.Sprintf("**Project**: `%s`\n\n---\n\n", cwd))

	for _, m := range messages {
		text := m.ExtractText()
		if text == "" {
			continue
		}
		switch m.Type {
		case claude.TypeUser:
			if m.IsExternalUser() {
				b.WriteString("## User\n\n")
			} else {
				b.WriteString("## User (system)\n\n")
			}
			b.WriteString(text)
			b.WriteString("\n\n")
		case claude.TypeAssistant:
			b.WriteString("## Assistant\n\n")
			b.WriteString(text)
			b.WriteString("\n\n---\n\n")
		}
	}

	output := b.String()
	if exportOutput != "" {
		return os.WriteFile(exportOutput, []byte(output), 0644)
	}
	fmt.Print(output)
	return nil
}
