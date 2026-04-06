package db

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(p, data, 0644); err != nil {
			return err
		}
	}

	// Remove stale extra files not in s.Files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // non-fatal
	}
	for _, e := range entries {
		if e.Name() == "SKILL.md" {
			continue
		}
		if _, ok := s.Files[e.Name()]; !ok {
			os.Remove(filepath.Join(dir, e.Name()))
		}
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
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || e.Name() == "SKILL.md" {
				continue
			}
			content, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			if s.Files == nil {
				s.Files = make(map[string][]byte)
			}
			s.Files[e.Name()] = content
		}
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
