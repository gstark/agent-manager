package db

type Skill struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Source      string   `yaml:"source"`
	Body        string   `yaml:"-"`
}

type Rule struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Paths       []string `yaml:"paths"`
	Body        string   `yaml:"-"`
}

type Pack struct {
	Name        string   `toml:"name"`
	Description string   `toml:"description"`
	Skills      []string `toml:"skills"`
	Rules       []string `toml:"rules"`
}
