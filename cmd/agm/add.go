package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/installer"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [skill|rule|policy|pack] <name>",
	Short: "Add a skill, rule, policy, or pack to the project config",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind, name := args[0], args[1]
		dir, _ := os.Getwd()

		cfg, err := config.LoadProjectConfig(dir)
		if err != nil {
			return fmt.Errorf("no %s found: %w", config.ProjectConfigFile, err)
		}

		switch kind {
		case "skill":
			if !slices.Contains(cfg.Skills, name) {
				cfg.Skills = append(cfg.Skills, name)
			}
		case "rule":
			if !slices.Contains(cfg.Rules, name) {
				cfg.Rules = append(cfg.Rules, name)
			}
		case "policy":
			if !slices.Contains(cfg.Policies, name) {
				cfg.Policies = append(cfg.Policies, name)
			}
		case "pack":
			if !slices.Contains(cfg.Packs, name) {
				cfg.Packs = append(cfg.Packs, name)
			}
		default:
			return fmt.Errorf("unknown type %q (use skill, rule, policy, or pack)", kind)
		}

		if err := config.SaveProjectConfig(dir, cfg); err != nil {
			return err
		}
		if _, err := installer.Install(dir, cfg); err != nil {
			return err
		}
		fmt.Printf("Added %s %q\n", kind, name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
