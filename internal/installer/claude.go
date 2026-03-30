package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gstark/agent-manager/internal/db"
)

func installClaude(projectDir string, r *resolved) error {
	// Write .claude/rules/*.md
	rulesDir := filepath.Join(projectDir, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return err
	}

	for _, rule := range r.rules {
		if err := writeClaudeRule(rulesDir, rule); err != nil {
			return err
		}
	}

	for _, lr := range r.localRules {
		rule := &db.Rule{
			Name:        lr.Name,
			Description: lr.Description,
			Paths:       lr.Paths,
			Body:        lr.Content,
		}
		if err := writeClaudeRule(rulesDir, rule); err != nil {
			return err
		}
	}

	// Write .claude/skills/*.md
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return err
	}

	for _, skill := range r.skills {
		content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n",
			skill.Name, skill.Description, strings.TrimSpace(skill.Body))
		dir := filepath.Join(skillsDir, skill.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
			return err
		}
	}

	// Create CLAUDE.md symlink
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	os.Remove(claudePath) // remove existing file/symlink
	if err := os.Symlink("AGENTS.md", claudePath); err != nil {
		return err
	}

	return nil
}

func writeClaudeRule(dir string, rule *db.Rule) error {
	var b strings.Builder
	b.WriteString("---\n")
	if len(rule.Paths) > 0 {
		b.WriteString("paths:\n")
		for _, p := range rule.Paths {
			fmt.Fprintf(&b, "  - %q\n", p)
		}
	}
	b.WriteString("---\n\n")
	if rule.Description != "" {
		fmt.Fprintf(&b, "# %s\n\n", rule.Description)
	}
	b.WriteString(strings.TrimSpace(rule.Body))
	b.WriteString("\n")

	return os.WriteFile(filepath.Join(dir, rule.Name+".md"), []byte(b.String()), 0644)
}
