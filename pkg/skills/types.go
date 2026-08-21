package skills

// Skill represents an Open Agent Skill defined in a SKILL.md file
type Skill struct {
	Name         string            `yaml:"name" json:"name"`
	Description  string            `yaml:"description" json:"description"`
	License      string            `yaml:"license,omitempty" json:"license,omitempty"`
	AllowedTools []string          `yaml:"allowed-tools,omitempty" json:"allowedTools,omitempty"`
	Metadata     map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Directory    string            `json:"directory"`
	FilePath     string            `json:"filePath"`
	Body         string            `json:"body"`
}

// Agent represents a ClaudeKit subagent definition in .claude/agents/*.md
type Agent struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	FilePath    string `json:"filePath"`
}

// Rule represents a development rule in .claude/rules/*.md
type Rule struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	FilePath string `json:"filePath"`
}

// Catalog holds all discovered skills, agents, and rules
type Catalog struct {
	Skills map[string]*Skill `json:"skills"`
	Agents map[string]*Agent `json:"agents"`
	Rules  map[string]*Rule  `json:"rules"`
}

// NewCatalog creates an empty catalog
func NewCatalog() *Catalog {
	return &Catalog{
		Skills: make(map[string]*Skill),
		Agents: make(map[string]*Agent),
		Rules:  make(map[string]*Rule),
	}
}
