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

var policiesCmd = &cobra.Command{
	Use:   "policies",
	Short: "Manage policies in the central database",
}

var policiesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all policies",
	RunE: func(cmd *cobra.Command, args []string) error {
		policies, err := db.ListPolicies()
		if err != nil {
			return err
		}
		if len(policies) == 0 {
			fmt.Println("No policies found. Create one with 'agm policies create <name>'.")
			return nil
		}

		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			type policyJSON struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			items := make([]policyJSON, len(policies))
			for i, p := range policies {
				items[i] = policyJSON{p.Name, p.Description}
			}
			return output.PrintJSON(items)
		}

		cols := []output.Column{
			{Name: "NAME", MinPct: 15, MaxPct: 35},
			{Name: "DESCRIPTION", MinPct: 30, MaxPct: 65},
		}
		rows := make([][]string, len(policies))
		for i, p := range policies {
			rows[i] = []string{p.Name, p.Description}
		}
		output.PrintTable(cols, rows)
		return nil
	},
}

var policiesCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new policy and open in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		p := &db.Policy{
			Name: name,
			Body: "Describe this policy here.",
		}
		if err := db.SavePolicy(p); err != nil {
			return err
		}
		path := config.PoliciesDir() + "/" + name + ".md"
		editor := getEditor()
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var policiesEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a policy in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.PoliciesDir() + "/" + name + ".md"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("policy %q not found", name)
		}
		editor := getEditor()
		c := exec.Command(editor, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var policiesCatCmd = &cobra.Command{
	Use:   "cat <name>",
	Short: "Print the contents of a policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := config.PoliciesDir() + "/" + name + ".md"
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("policy %q not found", name)
			}
			return err
		}
		fmt.Print(string(data))
		return nil
	},
}

var policiesDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a policy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := db.DeletePolicy(name); err != nil {
			return err
		}
		fmt.Printf("Deleted policy %q\n", name)
		return nil
	},
}

func init() {
	policiesListCmd.Flags().Bool("json", false, "Output as JSON (recommended for scripts and automation)")
	policiesCmd.AddCommand(policiesListCmd, policiesCreateCmd, policiesEditCmd, policiesCatCmd, policiesDeleteCmd)
	rootCmd.AddCommand(policiesCmd)
}
