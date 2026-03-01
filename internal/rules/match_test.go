package rules

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMatchRule(t *testing.T) {
	tests := []struct {
		name             string
		rule             Rule
		toolName         string
		command          string
		filePath         string
		url              string
		notificationType string
		wantMatch        bool
		wantErr          bool
	}{
		// --- tool matching ---
		{
			name:      "tool match single",
			rule:      Rule{Match: MatchConfig{Tool: StringOrList{"Bash"}}},
			toolName:  "Bash",
			wantMatch: true,
		},
		{
			name:      "tool no match",
			rule:      Rule{Match: MatchConfig{Tool: StringOrList{"Bash"}}},
			toolName:  "Read",
			wantMatch: false,
		},
		{
			name:      "tool match case insensitive",
			rule:      Rule{Match: MatchConfig{Tool: StringOrList{"bash"}}},
			toolName:  "Bash",
			wantMatch: true,
		},
		{
			name:      "tool match list",
			rule:      Rule{Match: MatchConfig{Tool: StringOrList{"Read", "Edit", "Write"}}},
			toolName:  "Edit",
			wantMatch: true,
		},
		{
			name:      "tool list no match",
			rule:      Rule{Match: MatchConfig{Tool: StringOrList{"Read", "Edit"}}},
			toolName:  "Bash",
			wantMatch: false,
		},
		{
			name:      "empty tool matches anything",
			rule:      Rule{Match: MatchConfig{}},
			toolName:  "Bash",
			wantMatch: true,
		},

		// --- command matching ---
		{
			name:      "command regex match",
			rule:      Rule{Match: MatchConfig{Command: `^git\s+`}},
			command:   "git status",
			wantMatch: true,
		},
		{
			name:      "command regex no match",
			rule:      Rule{Match: MatchConfig{Command: `^git\s+`}},
			command:   "ls -la",
			wantMatch: false,
		},
		{
			name:      "command partial match",
			rule:      Rule{Match: MatchConfig{Command: `rm\s+-rf`}},
			command:   "sudo rm -rf /",
			wantMatch: true,
		},

		// --- file_path matching ---
		{
			name:      "file_path regex match",
			rule:      Rule{Match: MatchConfig{FilePath: `\.go$`}},
			filePath:  "/Users/dev/main.go",
			wantMatch: true,
		},
		{
			name:      "file_path regex no match",
			rule:      Rule{Match: MatchConfig{FilePath: `\.go$`}},
			filePath:  "/Users/dev/main.py",
			wantMatch: false,
		},

		// --- url matching ---
		{
			name:      "url regex match",
			rule:      Rule{Match: MatchConfig{URL: `https://docs\.`}},
			url:       "https://docs.example.com/api",
			wantMatch: true,
		},
		{
			name:      "url regex no match",
			rule:      Rule{Match: MatchConfig{URL: `https://docs\.`}},
			url:       "https://example.com/api",
			wantMatch: false,
		},

		// --- notification_type matching ---
		{
			name:             "notification type match",
			rule:             Rule{Match: MatchConfig{NotificationType: "progress"}},
			notificationType: "progress",
			wantMatch:        true,
		},
		{
			name:             "notification type no match",
			rule:             Rule{Match: MatchConfig{NotificationType: "progress"}},
			notificationType: "error",
			wantMatch:        false,
		},
		{
			name:             "notification type regex",
			rule:             Rule{Match: MatchConfig{NotificationType: "^(progress|info)$"}},
			notificationType: "info",
			wantMatch:        true,
		},

		// --- AND logic ---
		{
			name: "AND all fields match",
			rule: Rule{Match: MatchConfig{
				Tool:    StringOrList{"Bash"},
				Command: `^git\s+`,
			}},
			toolName:  "Bash",
			command:   "git status",
			wantMatch: true,
		},
		{
			name: "AND tool matches but command doesn't",
			rule: Rule{Match: MatchConfig{
				Tool:    StringOrList{"Bash"},
				Command: `^git\s+`,
			}},
			toolName:  "Bash",
			command:   "ls -la",
			wantMatch: false,
		},
		{
			name: "AND command matches but tool doesn't",
			rule: Rule{Match: MatchConfig{
				Tool:    StringOrList{"Read"},
				Command: `^git\s+`,
			}},
			toolName:  "Bash",
			command:   "git status",
			wantMatch: false,
		},
		{
			name: "AND three fields all match",
			rule: Rule{Match: MatchConfig{
				Tool:     StringOrList{"Read", "Edit"},
				FilePath: `\.go$`,
				Command:  `main`,
			}},
			toolName:  "Edit",
			filePath:  "/foo/main.go",
			command:   "main",
			wantMatch: true,
		},
		{
			name: "AND three fields one fails",
			rule: Rule{Match: MatchConfig{
				Tool:     StringOrList{"Read", "Edit"},
				FilePath: `\.py$`,
				Command:  `main`,
			}},
			toolName:  "Edit",
			filePath:  "/foo/main.go",
			command:   "main",
			wantMatch: false,
		},

		// --- invalid regex ---
		{
			name:    "invalid command regex",
			rule:    Rule{Match: MatchConfig{Command: `[invalid`}},
			command: "anything",
			wantErr: true,
		},
		{
			name:     "invalid file_path regex",
			rule:     Rule{Match: MatchConfig{FilePath: `[invalid`}},
			filePath: "anything",
			wantErr:  true,
		},
		{
			name:    "invalid url regex",
			rule:    Rule{Match: MatchConfig{URL: `[invalid`}},
			url:     "anything",
			wantErr: true,
		},

		// --- empty values ---
		{
			name:      "no match criteria matches everything",
			rule:      Rule{Match: MatchConfig{}},
			toolName:  "Bash",
			command:   "rm -rf /",
			filePath:  "/etc/passwd",
			url:       "https://evil.com",
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchRule(&tt.rule, tt.toolName, tt.command, tt.filePath, tt.url, tt.notificationType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("matchRule() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantMatch {
				t.Errorf("matchRule() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestStringOrListUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []string
		wantErr bool
	}{
		{
			name: "single string",
			yaml: `tool: Bash`,
			want: []string{"Bash"},
		},
		{
			name: "list of strings",
			yaml: "tool:\n  - Read\n  - Edit\n  - Write",
			want: []string{"Read", "Edit", "Write"},
		},
		{
			name: "single string inline",
			yaml: `tool: "WebFetch"`,
			want: []string{"WebFetch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mc MatchConfig
			err := yaml.Unmarshal([]byte(tt.yaml), &mc)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(mc.Tool) != len(tt.want) {
				t.Fatalf("got %d tools, want %d: %v", len(mc.Tool), len(tt.want), mc.Tool)
			}
			for i, v := range mc.Tool {
				if v != tt.want[i] {
					t.Errorf("tool[%d] = %q, want %q", i, v, tt.want[i])
				}
			}
		})
	}
}

func TestRegexCache(t *testing.T) {
	// Call getRegex twice with same pattern to exercise the cache
	re1, err := getRegex(`^test\d+$`)
	if err != nil {
		t.Fatalf("getRegex() error = %v", err)
	}
	re2, err := getRegex(`^test\d+$`)
	if err != nil {
		t.Fatalf("getRegex() second call error = %v", err)
	}
	// Should be the same pointer (cached)
	if re1 != re2 {
		t.Error("expected cached regex to return same pointer")
	}

	// Invalid pattern
	_, err = getRegex(`[invalid`)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}
