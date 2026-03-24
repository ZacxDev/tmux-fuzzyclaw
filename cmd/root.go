package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zachatrocern/tmux-fuzzyclaw/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd is the base command for fuzzyclaw.
var rootCmd = &cobra.Command{
	Use:   "fuzzyclaw",
	Short: "Fuzzy task dashboard for tmux + Claude Code",
	Long:  "A TUI dashboard for managing Claude Code sessions across tmux windows.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load(cfgFile)
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to dashboard when no subcommand given
		return runDashboard(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/fuzzyclaw/config.yml)")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
