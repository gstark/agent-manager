package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize .agent-manager.toml in current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		path := config.ProjectConfigPath(dir)

		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", config.ProjectConfigFile)
		}

		cfg := &config.ProjectConfig{}
		if err := config.SaveProjectConfig(dir, cfg); err != nil {
			return err
		}

		// Append to .gitignore
		gitignorePath := filepath.Join(dir, ".gitignore")
		f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		f.WriteString("\n# agent-manager (generated)\n.claude/skills/\n.agents/skills/\n")

		fmt.Println("Created", config.ProjectConfigFile)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
