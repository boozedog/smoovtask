package rules

import (
	"testing"
)

func TestBashPipelineCheck(t *testing.T) {
	bp := NewBashPipeline(&BashPipelineConfig{
		SafeSinks:         []string{"head", "tail", "wc", "grep", "jq", "sort", "less"},
		GhAPIBlockedFlags: []string{"-X", "--method", "-f", "--field", "--raw-field"},
	})

	tests := []struct {
		name       string
		command    string
		wantDeny   bool
		wantReason string
	}{
		// --- sudo detection ---
		{
			name:       "sudo denied",
			command:    "sudo apt install vim",
			wantDeny:   true,
			wantReason: "sudo is not allowed",
		},
		{
			name:       "sudo in middle of command",
			command:    "echo hello && sudo rm -rf /",
			wantDeny:   true,
			wantReason: "sudo is not allowed",
		},
		{
			name:     "no sudo is fine",
			command:  "apt list --installed",
			wantDeny: false,
		},
		{
			name:     "pseudo is not sudo",
			command:  "echo pseudo",
			wantDeny: false,
		},

		// --- redirect detection ---
		{
			name:       "redirect to /etc",
			command:    "echo bad > /etc/hosts",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:       "append redirect to /etc",
			command:    "echo bad >> /etc/hosts",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:       "redirect to ~/.ssh/",
			command:    "echo key > ~/.ssh/authorized_keys",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:       "redirect to ~/.bashrc",
			command:    "echo alias > ~/.bashrc",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:       "redirect to /root/",
			command:    "echo data > /root/.profile",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:     "reading /etc is fine",
			command:  "cat /etc/hosts",
			wantDeny: false,
		},

		// --- command substitution detection ---
		{
			name:       "dollar-paren command substitution",
			command:    "echo $(whoami)",
			wantDeny:   true,
			wantReason: "command substitution is not allowed",
		},
		{
			name:       "backtick command substitution",
			command:    "echo `whoami`",
			wantDeny:   true,
			wantReason: "backtick command substitution is not allowed (use $() instead)",
		},

		// --- pipe chain validation ---
		{
			name:     "pipe to safe sink head",
			command:  "ls -la | head -20",
			wantDeny: false,
		},
		{
			name:     "pipe to safe sink grep",
			command:  "cat file.txt | grep pattern",
			wantDeny: false,
		},
		{
			name:     "pipe to safe sink jq",
			command:  "curl -s https://api.example.com | jq .",
			wantDeny: false,
		},
		{
			name:       "pipe to unknown sink denied",
			command:    "cat file.txt | some_unknown_command",
			wantDeny:   true,
			wantReason: "pipe to unknown command: some_unknown_command",
		},
		{
			name:       "pipe to xargs denied (removed from safe_sinks)",
			command:    "find . -name '*.go' | xargs grep TODO",
			wantDeny:   true,
			wantReason: "pipe to unknown command: xargs",
		},
		{
			name:     "no pipe is fine",
			command:  "ls -la",
			wantDeny: false,
		},

		// --- gh api mutation detection ---
		{
			name:       "gh api with -X flag",
			command:    "gh api repos/owner/repo -X POST",
			wantDeny:   true,
			wantReason: "gh api mutation flag blocked: -X",
		},
		{
			name:       "gh api with --method flag",
			command:    "gh api repos/owner/repo --method DELETE",
			wantDeny:   true,
			wantReason: "gh api mutation flag blocked: --method",
		},
		{
			name:       "gh api with -f flag",
			command:    "gh api repos/owner/repo -f title=hello",
			wantDeny:   true,
			wantReason: "gh api mutation flag blocked: -f",
		},
		{
			name:       "gh api with --field flag",
			command:    "gh api repos/owner/repo --field title=hello",
			wantDeny:   true,
			wantReason: "gh api mutation flag blocked: --field",
		},
		{
			name:       "gh api with --raw-field flag",
			command:    "gh api repos/owner/repo --raw-field body=test",
			wantDeny:   true,
			wantReason: "gh api mutation flag blocked: --raw-field",
		},
		{
			name:       "gh api with -XPOST joined flag",
			command:    "gh api repos/owner/repo -XPOST",
			wantDeny:   true,
			wantReason: "gh api mutation flag blocked: -X",
		},
		{
			name:       "gh api with --method=DELETE equals form",
			command:    "gh api repos/owner/repo --method=DELETE",
			wantDeny:   true,
			wantReason: "gh api mutation flag blocked: --method",
		},
		{
			name:     "gh api read only",
			command:  "gh api repos/owner/repo",
			wantDeny: false,
		},
		{
			name:     "gh not api command",
			command:  "gh pr list",
			wantDeny: false,
		},

		// --- semicolon chaining ---
		{
			name:       "semicolon chained with sudo",
			command:    "echo hello; sudo rm /tmp/file",
			wantDeny:   true,
			wantReason: "sudo is not allowed",
		},
		{
			name:       "semicolon chained with /etc write",
			command:    "echo foo; echo bad > /etc/hosts",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},

		// --- chained commands ---
		{
			name:       "chained with sudo in second part",
			command:    "echo hello && sudo rm /tmp/file",
			wantDeny:   true,
			wantReason: "sudo is not allowed",
		},
		{
			name:       "or-chained with /etc write",
			command:    "cat file || echo bad > /etc/hosts",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},

		// --- pipes inside quotes (should NOT be treated as shell pipes) ---
		{
			name:     "pipe in double-quoted go test -run regex",
			command:  `go test -v -run "TestA|TestB|TestC" ./...`,
			wantDeny: false,
		},
		{
			name:     "pipe in single-quoted grep pattern",
			command:  `grep 'foo|bar|baz' file.txt`,
			wantDeny: false,
		},
		{
			name:       "real pipe after quoted pipe chars",
			command:    `grep "foo|bar" file.txt | some_unknown`,
			wantDeny:   true,
			wantReason: "pipe to unknown command: some_unknown",
		},
		{
			name:     "real pipe to safe sink after quoted pipe chars",
			command:  `grep "foo|bar" file.txt | head -5`,
			wantDeny: false,
		},

		// --- combined ---
		{
			name:     "safe pipe chain in chained command",
			command:  "git log --oneline | head -10 && echo done",
			wantDeny: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deny, reason := bp.Check(tt.command)
			if deny != tt.wantDeny {
				t.Errorf("Check() deny = %v, want %v (reason: %q)", deny, tt.wantDeny, reason)
			}
			if tt.wantDeny && reason != tt.wantReason {
				t.Errorf("Check() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestBashPipelineNil(t *testing.T) {
	var bp *BashPipeline
	deny, reason := bp.Check("sudo rm -rf /")
	if deny {
		t.Errorf("nil pipeline should not deny, got deny=%v reason=%q", deny, reason)
	}
}

func TestNewBashPipelineNilConfig(t *testing.T) {
	bp := NewBashPipeline(nil)
	if bp != nil {
		t.Error("NewBashPipeline(nil) should return nil")
	}
}

func TestSplitChainedCommands(t *testing.T) {
	tests := []struct {
		name  string
		cmd   string
		count int
	}{
		{
			name:  "single command",
			cmd:   "ls -la",
			count: 1,
		},
		{
			name:  "two commands with &&",
			cmd:   "git add . && git commit -m test",
			count: 2,
		},
		{
			name:  "two commands with ||",
			cmd:   "cat file || echo fallback",
			count: 2,
		},
		{
			name:  "three commands mixed",
			cmd:   "cmd1 && cmd2 || cmd3",
			count: 3,
		},
		{
			name:  "semicolon separated",
			cmd:   "echo hello; echo world",
			count: 2,
		},
		{
			name:  "semicolon and && mixed",
			cmd:   "cmd1; cmd2 && cmd3",
			count: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitChainedCommands(tt.cmd)
			if len(parts) != tt.count {
				t.Errorf("splitChainedCommands(%q) = %d parts, want %d: %v", tt.cmd, len(parts), tt.count, parts)
			}
		})
	}
}

func TestSplitUnquoted(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  byte
		want []string
	}{
		{"no sep", "hello world", '|', []string{"hello world"}},
		{"simple pipe", "a | b", '|', []string{"a ", " b"}},
		{"pipe in double quotes", `go test -run "A|B" ./...`, '|', []string{`go test -run "A|B" ./...`}},
		{"pipe in single quotes", `grep 'a|b' f`, '|', []string{`grep 'a|b' f`}},
		{"real pipe after quoted", `grep "a|b" f | head`, '|', []string{`grep "a|b" f `, ` head`}},
		{"escaped quote", `echo "he said \"hi|bye\""`, '|', []string{`echo "he said \"hi|bye\""`}},
		{"backslash-escaped pipe", `echo a\|b`, '|', []string{`echo a\|b`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitUnquoted(tt.s, tt.sep)
			if len(got) != len(tt.want) {
				t.Fatalf("splitUnquoted(%q, %q) = %v (len %d), want %v (len %d)",
					tt.s, string(tt.sep), got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitUnquoted(%q, %q)[%d] = %q, want %q",
						tt.s, string(tt.sep), i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCheckDangerousOperators(t *testing.T) {
	tests := []struct {
		name       string
		part       string
		wantDeny   bool
		wantReason string
	}{
		{
			name:       "sudo",
			part:       "sudo rm -rf /",
			wantDeny:   true,
			wantReason: "sudo is not allowed",
		},
		{
			name:       "write to etc",
			part:       "echo bad > /etc/passwd",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:       "append to etc",
			part:       "echo bad >> /etc/passwd",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:       "redirect to ssh dir",
			part:       "echo key > ~/.ssh/authorized_keys",
			wantDeny:   true,
			wantReason: "redirect to sensitive path is not allowed",
		},
		{
			name:       "command substitution dollar-paren",
			part:       "echo $(id)",
			wantDeny:   true,
			wantReason: "command substitution is not allowed",
		},
		{
			name:       "command substitution backtick",
			part:       "echo `id`",
			wantDeny:   true,
			wantReason: "backtick command substitution is not allowed (use $() instead)",
		},
		{
			name:     "safe command",
			part:     "echo hello",
			wantDeny: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deny, reason := checkDangerousOperators(tt.part)
			if deny != tt.wantDeny {
				t.Errorf("checkDangerousOperators() deny = %v, want %v", deny, tt.wantDeny)
			}
			if tt.wantDeny && reason != tt.wantReason {
				t.Errorf("checkDangerousOperators() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestCheckDangerousOperatorsSmartCmdSub(t *testing.T) {
	// Build rulesets that allow git and echo commands.
	rulesets := []*Ruleset{
		{
			Name:     "test-allowlist",
			Priority: 50,
			Event:    "PreToolUse",
			Rules: []Rule{
				{
					Name:   "allow-git",
					Match:  MatchConfig{Tool: StringOrList{"Bash"}, Command: `^git\s+`},
					Action: ActionAllow,
				},
				{
					Name:   "allow-echo",
					Match:  MatchConfig{Tool: StringOrList{"Bash"}, Command: `^echo\b`},
					Action: ActionAllow,
				},
				{
					Name:   "allow-date",
					Match:  MatchConfig{Tool: StringOrList{"Bash"}, Command: `^date\b`},
					Action: ActionAllow,
				},
			},
		},
	}

	tests := []struct {
		name       string
		part       string
		wantDeny   bool
		wantReason string
	}{
		{
			name:     "allowed inner command: git branch",
			part:     "remote=$(git branch -r --list 'origin/main')",
			wantDeny: false,
		},
		{
			name:       "disallowed inner command: rm",
			part:       "result=$(rm -rf /)",
			wantDeny:   true,
			wantReason: "command substitution is not allowed",
		},
		{
			name:     "nested allowed commands",
			part:     "result=$(echo $(date))",
			wantDeny: false,
		},
		{
			name:       "nested with disallowed inner",
			part:       "result=$(echo $(rm -rf /))",
			wantDeny:   true,
			wantReason: "command substitution is not allowed",
		},
		{
			name:       "backticks still blocked even with rulesets",
			part:       "result=`git status`",
			wantDeny:   true,
			wantReason: "backtick command substitution is not allowed (use $() instead)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deny, reason := checkDangerousOperators(tt.part, rulesets)
			if deny != tt.wantDeny {
				t.Errorf("checkDangerousOperators() deny = %v, want %v (reason: %q)", deny, tt.wantDeny, reason)
			}
			if tt.wantDeny && reason != tt.wantReason {
				t.Errorf("checkDangerousOperators() reason = %q, want %q", reason, tt.wantReason)
			}
		})
	}
}

func TestExtractCommandSubstitutions(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []string
	}{
		{
			name: "no substitution",
			s:    "echo hello",
			want: nil,
		},
		{
			name: "simple substitution",
			s:    "echo $(date)",
			want: []string{"date"},
		},
		{
			name: "nested substitution",
			s:    "echo $(echo $(date))",
			want: []string{"echo $(date)", "date"},
		},
		{
			name: "multiple substitutions",
			s:    "echo $(date) $(whoami)",
			want: []string{"date", "whoami"},
		},
		{
			name: "complex inner command",
			s:    "remote=$(git branch -r --list 'origin/main')",
			want: []string{"git branch -r --list 'origin/main'"},
		},
		{
			name: "quoted parens not treated as nesting",
			s:    `x=$(echo "hello (world)")`,
			want: []string{`echo "hello (world)"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommandSubstitutions(tt.s)
			if len(got) != len(tt.want) {
				t.Fatalf("extractCommandSubstitutions(%q) = %v (len %d), want %v (len %d)",
					tt.s, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractCommandSubstitutions(%q)[%d] = %q, want %q",
						tt.s, i, got[i], tt.want[i])
				}
			}
		})
	}
}
