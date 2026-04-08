package db

type Skill struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Source      string            `yaml:"source"`
	ContentHash string            `yaml:"content_hash,omitempty"`
	Files       map[string][]byte `yaml:"-"`
	Body        string            `yaml:"-"`
}

type Rule struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Paths       []string `yaml:"paths"`
	Body        string   `yaml:"-"`
}

// Policy represents a Codex execution-policy rule, distinct from prompt
// instructions (Rule). Policies are installed into .codex/rules/.
type Policy struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Body        string `yaml:"-"`
}

type Pack struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Skills      []string `toml:"skills"`
	Rules       []string `toml:"rules"`
	Policies    []string `toml:"policies"`
}
