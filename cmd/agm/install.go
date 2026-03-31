package main

import (
	"fmt"
	"os"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/installer"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install skills and rules per .agent-manager.toml",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		cfg, err := config.LoadProjectConfig(dir)
		if err != nil {
			return fmt.Errorf("no %s found (run 'agm init' first): %w", config.ProjectConfigFile, err)
		}

		results, err := installer.Install(dir, cfg)
		if err != nil {
			return err
		}

		for _, r := range results {
			icon := "✓"
			if r.Status == installer.StatusUpToDate {
				icon = "·"
			}
			fmt.Printf("  %s %s: %s\n", icon, r.Kind, r.Name)
		}

		fmt.Println("\nAll items installed.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
