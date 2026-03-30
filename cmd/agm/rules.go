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

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage rules in the central database",
}

var rulesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		rules, err := db.ListRules()
		if err != nil {
			return err
		}
		if len(rules) == 0 {
			fmt.Println("No rules found. Create one with 'agm rules create <name>'.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tDESCRIPTION\tPATHS")
		for _, r := range rules {
			paths := strings.Join(r.Paths, ", ")
			if paths == "" {
				paths = "*"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, r.Description, paths)
		}
		return w.Flush()
	},
}

var rulesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		r := &db.Rule{
			Name: name,
			Body: "Describe this rule here.",
		}
		if err := db.SaveRule(r); err != nil {
			return err
		}
		fmt.Printf("Created rule %q. Edit with 'agm rules edit %s'.\n", name, name)
		return nil
	},
}

var rulesEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a rule in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.RulesDir() + "/" + name + ".md"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("rule %q not found", name)
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

var rulesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a rule",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := db.DeleteRule(name); err != nil {
			return err
		}
		fmt.Printf("Deleted rule %q\n", name)
		return nil
	},
}

func init() {
	rulesCmd.AddCommand(rulesListCmd, rulesCreateCmd, rulesEditCmd, rulesDeleteCmd)
	rootCmd.AddCommand(rulesCmd)
}
