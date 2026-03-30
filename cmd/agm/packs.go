package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
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
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSKILLS\tRULES")
		for _, p := range packs {
			fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name,
				strings.Join(p.Skills, ", "),
				strings.Join(p.Rules, ", "))
		}
		return w.Flush()
	},
}

var packsCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new pack",
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
		fmt.Printf("Created pack %q. Edit with 'agm packs edit %s'.\n", name, name)
		return nil
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
	packsCmd.AddCommand(packsListCmd, packsCreateCmd, packsEditCmd, packsDeleteCmd)
	rootCmd.AddCommand(packsCmd)
}
