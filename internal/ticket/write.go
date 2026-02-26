package ticket

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// frontmatterData is the YAML-serializable frontmatter structure.
// We use a separate struct to control field ordering and null handling.
type frontmatterData struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Project     string   `yaml:"project"`
	Status      Status   `yaml:"status"`
	PriorStatus *Status  `yaml:"prior-status"`
	Assignee    string   `yaml:"assignee"`
	Priority    Priority `yaml:"priority"`
	DependsOn   []string `yaml:"depends-on"`
	Created     string   `yaml:"created"`
	Updated     string   `yaml:"updated"`
	Tags        []string `yaml:"tags"`
}

// Render serializes a Ticket to markdown bytes (frontmatter + body).
// Alias: Marshal.
func Render(t *Ticket) ([]byte, error) {
	return Marshal(t)
}

// Marshal serializes a Ticket to markdown bytes (frontmatter + body).
func Marshal(t *Ticket) ([]byte, error) {
	fm := frontmatterData{
		ID:          t.ID,
		Title:       t.Title,
		Project:     t.Project,
		Status:      t.Status,
		PriorStatus: t.PriorStatus,
		Assignee:    t.Assignee,
		Priority:    t.Priority,
		DependsOn:   t.DependsOn,
		Created:     t.Created.UTC().Format(time.RFC3339),
		Updated:     t.Updated.UTC().Format(time.RFC3339),
		Tags:        t.Tags,
	}

	if fm.DependsOn == nil {
		fm.DependsOn = []string{}
	}
	if fm.Tags == nil {
		fm.Tags = []string{}
	}

	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.Write(yamlBytes)
	b.WriteString("---\n")

	if t.Body != "" {
		b.WriteString(t.Body)
	}

	return []byte(b.String()), nil
}

// AppendSection appends a new section to the ticket body.
// Format:
//
//	## <heading> — <timestamp>
//	**actor:** <actor> (session: <session>)
//	**key:** value
//
//	<content>
func AppendSection(t *Ticket, heading, actor, session, content string, fields map[string]string, ts time.Time) {
	var b strings.Builder

	// Ensure body ends with newline before appending
	if t.Body != "" && !strings.HasSuffix(t.Body, "\n") {
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "\n## %s — %s\n", heading, ts.UTC().Format(time.RFC3339))
	if actor != "" {
		if session != "" {
			fmt.Fprintf(&b, "**actor:** %s (session: %s)\n", actor, session)
		} else {
			fmt.Fprintf(&b, "**actor:** %s\n", actor)
		}
	}

	for k, v := range fields {
		fmt.Fprintf(&b, "**%s:** %s\n", k, v)
	}

	if content != "" {
		b.WriteString("\n")
		b.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			b.WriteString("\n")
		}
	}

	t.Body += b.String()
	t.Updated = ts
}
