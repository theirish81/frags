package frags

type Session struct {
	Prompt string `json:"prompt" yaml:"prompt"`
}

type Sessions map[string]*Session

type SessionManager struct {
	Prompt   string   `yaml:"prompt" json:"prompt"`
	Sessions Sessions `yaml:"sessions" json:"sessions"`
}
