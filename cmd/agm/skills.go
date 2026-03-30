package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/gstark/agent-manager/internal/output"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage skills in the central database",
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := db.ListSkills()
		if err != nil {
			return err
		}
		if len(skills) == 0 {
			fmt.Println("No skills found. Create one with 'agm skills create <name>'.")
			return nil
		}

		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			type skillJSON struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Source      string `json:"source"`
			}
			items := make([]skillJSON, len(skills))
			for i, s := range skills {
				src := s.Source
				if src == "" {
					src = "local"
				}
				items[i] = skillJSON{s.Name, s.Description, src}
			}
			return output.PrintJSON(items)
		}

		cols := []output.Column{
			{Name: "NAME", MinPct: 15, MaxPct: 30},
			{Name: "DESCRIPTION", MinPct: 30, MaxPct: 55},
			{Name: "SOURCE", MinPct: 10, MaxPct: 30},
		}
		rows := make([][]string, len(skills))
		for i, s := range skills {
			source := s.Source
			if source == "" {
				source = "local"
			}
			rows[i] = []string{s.Name, s.Description, source}
		}
		output.PrintTable(cols, rows)
		return nil
	},
}

var skillsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := &db.Skill{
			Name:   name,
			Source: "local",
			Body:   "# " + name + "\n\nDescribe this skill here.",
		}
		if err := db.SaveSkill(s); err != nil {
			return err
		}
		fmt.Printf("Created skill %q. Edit with 'agm skills edit %s'.\n", name, name)
		return nil
	},
}

var skillsEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a skill in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.SkillsDir() + "/" + name + ".md"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("skill %q not found", name)
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var skillsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a skill",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := db.DeleteSkill(name); err != nil {
			return err
		}
		fmt.Printf("Deleted skill %q\n", name)
		return nil
	},
}

func init() {
	skillsListCmd.Flags().Bool("json", false, "Output as JSON")
	skillsCmd.AddCommand(skillsListCmd, skillsCreateCmd, skillsEditCmd, skillsDeleteCmd)
	rootCmd.AddCommand(skillsCmd)
}
