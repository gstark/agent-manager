package installer

import (
	"fmt"

	"github.com/gstark/agent-manager/internal/config"
	"github.com/gstark/agent-manager/internal/db"
)

// ItemStatus indicates whether an item was installed fresh or already current.
type ItemStatus int

const (
	StatusInstalled ItemStatus = iota
	StatusUpToDate
)

// ItemResult describes the outcome of installing a single item.
type ItemResult struct {
	Kind   string // "skill" or "rule"
	Name   string
	Status ItemStatus
}

// ArtifactSet is the neutral, tool-agnostic output of artifact resolution.
// Each ToolAdapter receives the full set and consumes only what it understands.
type ArtifactSet struct {
	Skills     []*db.Skill
	Rules      []*db.Rule
	Policies   []*db.Policy
	LocalRules []config.LocalRule
}

// ToolAdapter installs artifacts for a specific tool runtime.
type ToolAdapter interface {
	Name() string
	Install(projectDir string, artifacts *ArtifactSet) ([]ItemResult, error)
}

// DefaultAdapters is the set of adapters used by Install.
// Append to this slice to register a third-party tool adapter.
var DefaultAdapters = []ToolAdapter{ClaudeAdapter{}, CodexAdapter{}}

func resolve(cfg *config.ProjectConfig) (*ArtifactSet, error) {
	r := &ArtifactSet{LocalRules: cfg.LocalRules}
	seen := struct {
		skills   map[string]bool
		rules    map[string]bool
		policies map[string]bool
	}{make(map[string]bool), make(map[string]bool), make(map[string]bool)}

	addSkill := func(name string) error {
		if seen.skills[name] {
			return nil
		}
		s, err := db.LoadSkill(name)
		if err != nil {
			return fmt.Errorf("skill %q: %w", name, err)
		}
		r.Skills = append(r.Skills, s)
		seen.skills[name] = true
		return nil
	}

	addRule := func(name string) error {
		if seen.rules[name] {
			return nil
		}
		rule, err := db.LoadRule(name)
		if err != nil {
			return fmt.Errorf("rule %q: %w", name, err)
		}
		r.Rules = append(r.Rules, rule)
		seen.rules[name] = true
		return nil
	}

	addPolicy := func(name string) error {
		if seen.policies[name] {
			return nil
		}
		p, err := db.LoadPolicy(name)
		if err != nil {
			return fmt.Errorf("policy %q: %w", name, err)
		}
		r.Policies = append(r.Policies, p)
		seen.policies[name] = true
		return nil
	}

	// Expand packs first
	for _, packName := range cfg.Packs {
		p, err := db.LoadPack(packName)
		if err != nil {
			return nil, fmt.Errorf("pack %q: %w", packName, err)
		}
		for _, s := range p.Skills {
			if err := addSkill(s); err != nil {
				return nil, err
			}
		}
		for _, r := range p.Rules {
			if err := addRule(r); err != nil {
				return nil, err
			}
		}
		for _, pol := range p.Policies {
			if err := addPolicy(pol); err != nil {
				return nil, err
			}
		}
	}

	// Then explicit skills/rules/policies (dedup against packs)
	for _, name := range cfg.Skills {
		if err := addSkill(name); err != nil {
			return nil, err
		}
	}
	for _, name := range cfg.Rules {
		if err := addRule(name); err != nil {
			return nil, err
		}
	}
	for _, name := range cfg.Policies {
		if err := addPolicy(name); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// Install resolves artifacts and dispatches to DefaultAdapters.
func Install(projectDir string, cfg *config.ProjectConfig) ([]ItemResult, error) {
	return InstallWith(projectDir, cfg, DefaultAdapters)
}

// InstallWith resolves artifacts and dispatches to the given adapters.
func InstallWith(projectDir string, cfg *config.ProjectConfig, adapters []ToolAdapter) ([]ItemResult, error) {
	a, err := resolve(cfg)
	if err != nil {
		return nil, err
	}

	// Install skills once to canonical .agm/skills/ tree
	results, err := installCanonicalSkills(projectDir, a.Skills)
	if err != nil {
		return nil, fmt.Errorf("canonical skills: %w", err)
	}

	for _, adapter := range adapters {
		ar, err := adapter.Install(projectDir, a)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", adapter.Name(), err)
		}
		results = append(results, ar...)
	}

	return results, nil
}
