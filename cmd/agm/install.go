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

		if err := installer.Install(dir, cfg); err != nil {
			return err
		}

		fmt.Println("Installed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
