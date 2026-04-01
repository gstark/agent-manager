package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/gstark/agent-manager/internal/output"
	"github.com/spf13/cobra"
)

var packsCmd = &cobra.Command{
	Use:   "packs",
	Short: "Manage packs in the central database",
}

var packsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packs",
	RunE: func(cmd *cobra.Command, args []string) error {
		packs, err := db.ListPacks()
		if err != nil {
			return err
		}
		if len(packs) == 0 {
			fmt.Println("No packs found. Create one with 'agm packs create <name>'.")
			return nil
		}

		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			type packJSON struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Skills      []string `json:"skills"`
				Rules       []string `json:"rules"`
			}
			items := make([]packJSON, len(packs))
			for i, p := range packs {
				items[i] = packJSON{p.Name, p.Description, p.Skills, p.Rules}
			}
			return output.PrintJSON(items)
		}

		cols := []output.Column{
			{Name: "NAME", MinPct: 10, MaxPct: 25},
			{Name: "SKILLS", MinPct: 15, MaxPct: 35},
			{Name: "RULES", MinPct: 20, MaxPct: 50},
		}
		rows := make([][]string, len(packs))
		for i, p := range packs {
			skills := strings.Join(p.Skills, ", ")
			if skills == "" {
				skills = "—"
			}
			rules := strings.Join(p.Rules, ", ")
			if rules == "" {
				rules = "—"
			}
			rows[i] = []string{p.Name, skills, rules}
		}
		output.PrintTable(cols, rows)
		return nil
	},
}

var packsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new pack and open in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		p := &db.Pack{
			Name:        name,
			Description: name + " pack",
		}
		if err := db.SavePack(p); err != nil {
			return err
		}
		path := config.PacksDir() + "/" + name + ".toml"
		editor := getEditor()
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var packsEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a pack in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.PacksDir() + "/" + name + ".toml"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("pack %q not found", name)
		}
		editor := getEditor()
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var packsDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a pack",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := db.DeletePack(name); err != nil {
			return err
		}
		fmt.Printf("Deleted pack %q\n", name)
		return nil
	},
}

func init() {
	packsListCmd.Flags().Bool("json", false, "Output as JSON (recommended for scripts and automation)")
	packsCmd.AddCommand(packsListCmd, packsCreateCmd, packsEditCmd, packsDeleteCmd)
	rootCmd.AddCommand(packsCmd)
}
