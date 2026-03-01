package rules

import "gopkg.in/yaml.v3"

// Action represents the decision for a matched rule.
type Action string

const (
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
	ActionAsk   Action = "ask"
)

// MatchConfig defines pattern matching criteria for a rule. All fields
// are optional and AND-ed together.
type MatchConfig struct {
	Tool             StringOrList `yaml:"tool"`
	Command          string       `yaml:"command"`
	FilePath         string       `yaml:"file_path"`
	URL              string       `yaml:"url"`
	NotificationType string       `yaml:"notification_type"`
}

// StringOrList allows a YAML field to be either a single string or a list of strings.
type StringOrList []string

func (s *StringOrList) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*s = []string{value.Value}
		return nil
	}
	var list []string
	if err := value.Decode(&list); err != nil {
		return err
	}
	*s = list
	return nil
}

// Rule is a single matching rule within a ruleset.
type Rule struct {
	Name    string      `yaml:"name"`
	Match   MatchConfig `yaml:"match"`
	Action  Action      `yaml:"action"`
	Message string      `yaml:"message"`
}

// Ruleset is a collection of rules loaded from a single YAML file.
type Ruleset struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Priority    int    `yaml:"priority"`
	Event       string `yaml:"event"`
	Type        string `yaml:"type"` // empty for normal rulesets, "bash-pipeline" for bash config
	Rules       []Rule `yaml:"rules"`

	// BashPipeline config (only when Type == "bash-pipeline")
	Config *BashPipelineConfig `yaml:"config,omitempty"`
}

// BashPipelineConfig holds configuration for the bash structural analysis.
type BashPipelineConfig struct {
	SafeSinks         []string `yaml:"safe_sinks"`
	GhAPIBlockedFlags []string `yaml:"gh_api_blocked_flags"`
}

// EvalResult holds the outcome of rule evaluation.
type EvalResult struct {
	Decision Action
	Reason   string
	Ruleset  string
	Rule     string
}
