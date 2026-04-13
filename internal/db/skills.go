package db

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/gstark/agent-manager/internal/config"
	"gopkg.in/yaml.v3"
)

func skillDir(name string) string {
	return filepath.Join(config.SkillsDir(), name)
}

func skillPath(name string) string {
	return filepath.Join(skillDir(name), "SKILL.md")
}

func SaveSkill(s *Skill) error {
	dir := skillDir(s.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	fm, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), strings.TrimSpace(s.Body))
	if err := os.WriteFile(skillPath(s.Name), []byte(content), 0644); err != nil {
		return err
	}

	// Write extra files
	for name, data := range s.Files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0644); err != nil {
			return err
		}
	}

	if err := removeStaleSkillFiles(dir, s.Files); err != nil {
		return err
	}

	return nil
}

func LoadSkill(name string) (*Skill, error) {
	data, err := os.ReadFile(skillPath(name))
	if err != nil {
		return nil, err
	}
	s := &Skill{}
	rest, err := frontmatter.Parse(bytes.NewReader(data), s)
	if err != nil {
		return nil, err
	}
	s.Body = strings.TrimSpace(string(rest))

	// Load extra files
	dir := skillDir(name)
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "SKILL.md" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if s.Files == nil {
			s.Files = make(map[string][]byte)
		}
		s.Files[rel] = content
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return s, nil
}

func ListSkills() ([]*Skill, error) {
	entries, err := os.ReadDir(config.SkillsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var skills []*Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := LoadSkill(e.Name())
		if err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

func DeleteSkill(name string) error {
	return os.RemoveAll(skillDir(name))
}

func removeStaleSkillFiles(dir string, wantedFiles map[string][]byte) error {
	wanted := make(map[string]bool, len(wantedFiles))
	for name := range wantedFiles {
		wanted[filepath.ToSlash(filepath.Clean(name))] = true
	}

	var dirs []string
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if rel == "." {
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if rel == "SKILL.md" || wanted[rel] {
			return nil
		}
		return os.Remove(path)
	}); err != nil {
		return err
	}

	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	for _, path := range dirs {
		entries, err := os.ReadDir(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if len(entries) == 0 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}
