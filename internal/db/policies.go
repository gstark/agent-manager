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

func policyPath(name string) string {
	return filepath.Join(config.PoliciesDir(), name+".md")
}

func SavePolicy(p *Policy) error {
	fm, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), strings.TrimSpace(p.Body))
	return os.WriteFile(policyPath(p.Name), []byte(content), 0644)
}

func LoadPolicy(name string) (*Policy, error) {
	data, err := os.ReadFile(policyPath(name))
	if err != nil {
		return nil, err
	}
	p := &Policy{}
	rest, err := frontmatter.Parse(bytes.NewReader(data), p)
	if err != nil {
		return nil, err
	}
	p.Body = strings.TrimSpace(string(rest))
	return p, nil
}

func ListPolicies() ([]*Policy, error) {
	entries, err := os.ReadDir(config.PoliciesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var policies []*Policy
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		p, err := LoadPolicy(name)
		if err != nil {
			continue
		}
		policies = append(policies, p)
	}
	return policies, nil
}

func DeletePolicy(name string) error {
	return os.Remove(policyPath(name))
}
