package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gstark/agent-manager/internal/db"
)

// writeFileIfChanged writes content to path and returns true if the file was
// created or its content changed.
func writeFileIfChanged(path string, content []byte) (changed bool, err error) {
	existing, err := os.ReadFile(path)
	if err == nil && string(existing) == string(content) {
		return false, nil
	}
	return true, os.WriteFile(path, content, 0644)
}

func installClaude(projectDir string, r *resolved) ([]ItemResult, error) {
	var results []ItemResult

	// Write .claude/rules/*.md
	rulesDir := filepath.Join(projectDir, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return nil, err
	}

	for _, rule := range r.rules {
		changed, err := writeClaudeRule(rulesDir, rule)
		if err != nil {
			return nil, err
		}
		status := StatusUpToDate
		if changed {
			status = StatusInstalled
		}
		results = append(results, ItemResult{Kind: "rule", Name: rule.Name, Status: status})
	}

	for _, lr := range r.localRules {
		rule := &db.Rule{
			Name:        lr.Name,
			Description: lr.Description,
			Paths:       lr.Paths,
			Body:        lr.Content,
		}
		changed, err := writeClaudeRule(rulesDir, rule)
		if err != nil {
			return nil, err
		}
		status := StatusUpToDate
		if changed {
			status = StatusInstalled
		}
		results = append(results, ItemResult{Kind: "rule", Name: rule.Name, Status: status})
	}

	// Write .claude/skills/*.md
	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return nil, err
	}

	for _, skill := range r.skills {
		content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n",
			skill.Name, skill.Description, strings.TrimSpace(skill.Body))
		dir := filepath.Join(skillsDir, skill.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
		changed, err := writeFileIfChanged(filepath.Join(dir, "SKILL.md"), []byte(content))
		if err != nil {
			return nil, err
		}
		status := StatusUpToDate
		if changed {
			status = StatusInstalled
		}
		results = append(results, ItemResult{Kind: "skill", Name: skill.Name, Status: status})
	}

	// Create CLAUDE.md symlink
	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	os.Remove(claudePath) // remove existing file/symlink
	if err := os.Symlink("AGENTS.md", claudePath); err != nil {
		return nil, err
	}

	return results, nil
}

func writeClaudeRule(dir string, rule *db.Rule) (changed bool, err error) {
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

	return writeFileIfChanged(filepath.Join(dir, rule.Name+".md"), []byte(b.String()))
}
