package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [skill|rule|pack] <name>",
	Short: "Remove a skill, rule, or pack from the project config",
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
			cfg.Skills = slices.DeleteFunc(cfg.Skills, func(s string) bool { return s == name })
		case "rule":
			cfg.Rules = slices.DeleteFunc(cfg.Rules, func(s string) bool { return s == name })
		case "pack":
			cfg.Packs = slices.DeleteFunc(cfg.Packs, func(s string) bool { return s == name })
		default:
			return fmt.Errorf("unknown type %q (use skill, rule, or pack)", kind)
		}

		if err := config.SaveProjectConfig(dir, cfg); err != nil {
			return err
		}
		fmt.Printf("Removed %s %q\n", kind, name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
