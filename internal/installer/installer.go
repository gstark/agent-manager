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

type resolved struct {
	skills     []*db.Skill
	rules      []*db.Rule
	localRules []config.LocalRule
}

func resolve(cfg *config.ProjectConfig) (*resolved, error) {
	r := &resolved{localRules: cfg.LocalRules}
	seen := struct {
		skills map[string]bool
		rules  map[string]bool
	}{make(map[string]bool), make(map[string]bool)}

	addSkill := func(name string) error {
		if seen.skills[name] {
			return nil
		}
		s, err := db.LoadSkill(name)
		if err != nil {
			return fmt.Errorf("skill %q: %w", name, err)
		}
		r.skills = append(r.skills, s)
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
		r.rules = append(r.rules, rule)
		seen.rules[name] = true
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
	}

	// Then explicit skills/rules (dedup against packs)
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

	return r, nil
}

func Install(projectDir string, cfg *config.ProjectConfig) ([]ItemResult, error) {
	r, err := resolve(cfg)
	if err != nil {
		return nil, err
	}

	results, err := installClaude(projectDir, r)
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}
	if err := installCodex(projectDir, r); err != nil {
		return nil, fmt.Errorf("codex: %w", err)
	}
	return results, nil
}
