package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [skill|rule|pack] <name>",
	Short: "Add a skill, rule, or pack to the project config",
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
		case "pack":
			if !slices.Contains(cfg.Packs, name) {
				cfg.Packs = append(cfg.Packs, name)
			}
		default:
			return fmt.Errorf("unknown type %q (use skill, rule, or pack)", kind)
		}

		if err := config.SaveProjectConfig(dir, cfg); err != nil {
			return err
		}
		fmt.Printf("Added %s %q\n", kind, name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
