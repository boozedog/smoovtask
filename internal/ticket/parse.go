package ticket

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse parses a markdown ticket file content into a Ticket.
// The format is:
//
//	---
//	<yaml frontmatter>
//	---
//
//	<markdown body>
func Parse(data []byte) (*Ticket, error) {
	content := string(data)

	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}

	t, err := parseFrontmatter(frontmatter)
	if err != nil {
		return nil, err
	}
	t.Body = body

	return t, nil
}

// ParseFrontmatter parses only the YAML frontmatter section into a Ticket.
// Body is intentionally left empty.
func ParseFrontmatter(data []byte) (*Ticket, error) {
	content := string(data)
	frontmatter, _, err := splitFrontmatter(content)
	if err != nil {
		return nil, err
	}
	return parseFrontmatter(frontmatter)
}

func parseFrontmatter(frontmatter string) (*Ticket, error) {
	var t Ticket
	if err := yaml.Unmarshal([]byte(frontmatter), &t); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return &t, nil
}

// splitFrontmatter splits markdown content into frontmatter YAML and body.
// Expects --- on the first line and a closing --- on its own line.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("missing opening frontmatter delimiter")
	}

	// Find the closing ---
	// Skip the first line (the opening ---)
	rest := content[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	before, after, found := strings.Cut(rest, "\n---")
	if !found {
		return "", "", fmt.Errorf("missing closing frontmatter delimiter")
	}

	frontmatter = before
	if len(after) > 0 && after[0] == '\n' {
		after = after[1:]
	}

	return frontmatter, after, nil
}
