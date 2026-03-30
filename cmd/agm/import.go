package main

import (
	"fmt"

	"github.com/gstark/agent-manager/internal/db"
	"github.com/gstark/agent-manager/internal/importer"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import <owner/repo@skill>",
	Short: "Import a skill from skills.sh",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := importer.ParseSkillRef(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Fetching %s...\n", ref.Source())
		skill, err := importer.Import(ref)
		if err != nil {
			return err
		}

		if err := db.SaveSkill(skill); err != nil {
			return err
		}

		fmt.Printf("Imported skill %q from %s\n", skill.Name, skill.Source)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
}
