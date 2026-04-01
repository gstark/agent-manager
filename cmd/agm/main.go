package main

import (
	"fmt"
	"os"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "agm",
	Short: "Agent Manager — manage AI agent configurations",
	Long: "A CLI and TUI tool for managing skills, rules, and packs across Claude Code and Codex.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureDirs()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("agm", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// getEditor returns the user's preferred editor: $VISUAL, $EDITOR, or "vim".
func getEditor() string {
	if e := os.Getenv("VISUAL"); e != "" {
		return e
	}
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vim"
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
