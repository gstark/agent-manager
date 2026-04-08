package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gstark/agent-manager/internal/db"
)

// installCanonicalSkills writes each skill once to .agm/skills/<name>/ and
// returns per-skill install status.
func installCanonicalSkills(projectDir string, skills []*db.Skill) ([]ItemResult, error) {
	var results []ItemResult
	skillsDir := filepath.Join(projectDir, ".agm", "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return nil, err
	}

	for _, skill := range skills {
		dir := filepath.Join(skillsDir, skill.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\n%s\n",
			skill.Name, skill.Description, strings.TrimSpace(skill.Body))
		changed, err := writeFileIfChanged(filepath.Join(dir, "SKILL.md"), []byte(content))
		if err != nil {
			return nil, err
		}

		for name, data := range skill.Files {
			c, writeErr := writeFileIfChanged(filepath.Join(dir, name), data)
			if writeErr != nil {
				return nil, writeErr
			}
			if c {
				changed = true
			}
		}

		status := StatusUpToDate
		if changed {
			status = StatusInstalled
		}
		results = append(results, ItemResult{Kind: "skill", Name: skill.Name, Status: status})
	}

	return results, nil
}

// projectSkills creates symlinks from targetBase/skills/<name> -> .agm/skills/<name>.
// Falls back to copying if symlink creation fails.
func projectSkills(projectDir string, targetBase string, skills []*db.Skill) error {
	targetDir := filepath.Join(projectDir, targetBase, "skills")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	for _, skill := range skills {
		linkPath := filepath.Join(targetDir, skill.Name)
		// Relative path from targetDir to canonical dir
		// e.g. from .claude/skills/tdd -> ../../.agm/skills/tdd
		relTarget := filepath.Join("..", "..", ".agm", "skills", skill.Name)

		// Remove existing entry (file, dir, or symlink)
		if err := os.RemoveAll(linkPath); err != nil {
			return err
		}

		if err := os.Symlink(relTarget, linkPath); err != nil {
			// Fallback: copy the canonical directory
			if copyErr := copyDir(
				filepath.Join(projectDir, ".agm", "skills", skill.Name),
				linkPath,
			); copyErr != nil {
				return fmt.Errorf("symlink failed (%w) and copy fallback failed: %w", err, copyErr)
			}
		}
	}

	return nil
}

// copyDir recursively copies src to dst.
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())

		if e.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return err
		}
	}

	return nil
}
