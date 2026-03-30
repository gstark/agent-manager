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

func rulePath(name string) string {
	return filepath.Join(config.RulesDir(), name+".md")
}

func SaveRule(r *Rule) error {
	fm, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), strings.TrimSpace(r.Body))
	return os.WriteFile(rulePath(r.Name), []byte(content), 0644)
}

func LoadRule(name string) (*Rule, error) {
	data, err := os.ReadFile(rulePath(name))
	if err != nil {
		return nil, err
	}
	r := &Rule{}
	rest, err := frontmatter.Parse(bytes.NewReader(data), r)
	if err != nil {
		return nil, err
	}
	r.Body = strings.TrimSpace(string(rest))
	return r, nil
}

func ListRules() ([]*Rule, error) {
	entries, err := os.ReadDir(config.RulesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rules []*Rule
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		r, err := LoadRule(name)
		if err != nil {
			continue
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func DeleteRule(name string) error {
	return os.Remove(rulePath(name))
}
