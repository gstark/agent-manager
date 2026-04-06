package main

import (
	"fmt"
	"strings"

	"github.com/gstark/agent-manager/internal/auditor"
	"github.com/gstark/agent-manager/internal/db"
	"github.com/gstark/agent-manager/internal/importer"
	"github.com/gstark/agent-manager/internal/output"
	"github.com/spf13/cobra"
)

var skillsAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Check imported skills for upstream changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		skills, err := db.ListSkills()
		if err != nil {
			return err
		}

		fetch := func(source string) (string, map[string][]byte, error) {
			raw := strings.TrimPrefix(source, "skills.sh/")
			ref, err := importer.ParseSkillRef(raw)
			if err != nil {
				return "", nil, err
			}
			s, err := importer.Import(ref)
			if err != nil {
				return "", nil, err
			}
			return s.Body, s.Files, nil
		}

		results := auditor.Audit(skills, fetch)

		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			type resultJSON struct {
				Name   string `json:"name"`
				Source string `json:"source"`
				Status string `json:"status"`
				Error  string `json:"error,omitempty"`
			}
			items := make([]resultJSON, len(results))
			for i, r := range results {
				items[i] = resultJSON{r.Name, r.Source, string(r.Status), r.Error}
			}
			return output.PrintJSON(items)
		}

		if len(results) > 0 {
			cols := []output.Column{
				{Name: "NAME", MinPct: 15, MaxPct: 30},
				{Name: "SOURCE", MinPct: 20, MaxPct: 40},
				{Name: "STATUS", MinPct: 10, MaxPct: 30},
			}
			rows := make([][]string, len(results))
			for i, r := range results {
				status := string(r.Status)
				if r.Error != "" {
					status += ": " + r.Error
				}
				rows[i] = []string{r.Name, r.Source, status}
			}
			output.PrintTable(cols, rows)
		}

		// Summary
		var upToDate, changed, errors int
		for _, r := range results {
			switch r.Status {
			case auditor.StatusUpToDate:
				upToDate++
			case auditor.StatusChanged:
				changed++
			case auditor.StatusError:
				errors++
			}
		}
		fmt.Printf("%d checked, %d up-to-date, %d changed, %d errors\n",
			len(results), upToDate, changed, errors)

		return nil
	},
}

func init() {
	skillsAuditCmd.Flags().Bool("json", false, "Output as JSON")
	skillsCmd.AddCommand(skillsAuditCmd)
}
