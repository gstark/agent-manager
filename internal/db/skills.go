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

func skillPath(name string) string {
	return filepath.Join(config.SkillsDir(), name+".md")
}

func SaveSkill(s *Skill) error {
	fm, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), strings.TrimSpace(s.Body))
	return os.WriteFile(skillPath(s.Name), []byte(content), 0644)
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
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		s, err := LoadSkill(name)
		if err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

func DeleteSkill(name string) error {
	return os.Remove(skillPath(name))
}
