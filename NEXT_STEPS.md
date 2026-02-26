# smoovtask — Next Steps

## What's Done (Phases 1–3)

The core CLI is complete. All commands, hooks, workflow engine, and tests are implemented.

### Commands (16 total)
- `init`, `new`, `list`, `show`, `pick`, `note`, `status`, `review`
- `assign`, `hold`, `unhold`, `close`, `override`, `context`
- `hook <event>` (10 handlers), `hooks install`

### Internals
- Workflow state machine with transition rules, review eligibility, dependency auto-blocking/unblocking
- Markdown tickets with YAML frontmatter in Obsidian vault
- JSONL event log with daily rotation and flock-based append
- Project detection from PWD, session identity from env vars
- Priority-scored batch selection in session-start hook

### Test Coverage
| Package | Coverage |
|---------|----------|
| `cmd/` | 78% |
| `internal/ticket/` | 85% |
| `internal/event/` | 82% |
| `internal/hook/` | 66% |
| `internal/workflow/` | 96% |
| `internal/identity/` | 100% |
| `internal/project/` | 100% |
| `internal/config/` | 35% |

---

## What's Left (from DESIGN.md)

### Phase 4: TUI — `st board`

Terminal kanban board using bubbletea + lipgloss + bubbles.

**Views:**
- **List view (default)** — tickets grouped by project, sorted by priority
- **Kanban view** — columns for BACKLOG, OPEN, IN-PROGRESS, REVIEW, DONE

**Features:**
- Keyboard navigation, filtering by project/status/priority
- Toggle between list and kanban views
- Real-time: tail JSONL and refresh on new events

**Packages:**
- `internal/tui/` — shared model, styles
- `internal/tui/board/` — kanban view
- `internal/tui/list/` — list view
- `cmd/board.go` — cobra command

**Estimated scope:** ~500–800 lines. Moderate complexity — bubbletea has a learning curve but the data layer is done.

### Phase 5: Web UI — `st web`

Web server embedded in the `st` binary.

**Tech stack:**
- Go net/http server
- templ (type-safe Go HTML templates)
- htmx for dynamic interactions
- Franken UI (UIKit + Tailwind) for styling
- SSE for real-time updates (htmx SSE extension)
- Static assets embedded via `go:embed`

**Views:**
- Kanban board (cards move in real-time via SSE)
- List view (sortable/filterable)
- Ticket detail (rendered markdown, dependency links)
- Activity feed (live JSONL stream)

**Packages:**
- `internal/web/handler/` — HTTP handlers
- `internal/web/sse/` — SSE event streaming (tails JSONL)
- `internal/web/templates/` — templ templates
- `internal/web/static/` — embedded CSS/JS assets
- `cmd/web.go` — cobra command

**Estimated scope:** ~1500–2500 lines. Largest remaining feature. SSE streaming from JSONL tail is the interesting part.

### Phase 6: Plugin System

Extension mechanism for user-specific behaviors triggered by events.

**Core:**
- `internal/plugin/` — plugin loading, config parsing, event fan-out
- Glob-based event matching (`"status.*"`, `"hook.permission-request"`)
- Per-event invocation (stateless — plugins are short-lived processes)
- JSON on stdin (same format as JSONL events), optional JSON on stdout

**Config (in config.toml):**
```toml
[[plugins]]
name = "audio-alerts"
enabled = true
command = "st-plugin-audio"
events = ["status.*", "ticket.created"]

[plugins.config]
player = "afplay"
tts_command = "say"
```

**Example plugins (ship as scripts in repo):**
- **auto-approve** — auto-approve Claude Code permission requests when agent has active ticket
- **audio-alerts** — system sounds + TTS on ticket transitions (macOS: afplay/say, Linux: paplay/espeak)
- **slack-notify** — webhook to Slack on status changes
- **observability** — POST event payloads to external endpoints

**Estimated scope:** ~300–500 lines for core, ~100 lines per example plugin.

### Smaller Items

| Item | Description | Scope |
|------|-------------|-------|
| `st gc` | Garbage collection for old JSONL event logs | Small (~50 lines) |
| `cmd/init_test.go` | Missing test for `st init` command | Small |
| Config test coverage | `internal/config/` at 35% — room for improvement | Small–medium |
| Hook test coverage | `internal/hook/` at 66% — session_start has complex logic | Medium |

---

## Suggested Order

1. **Phase 4: TUI** — immediate value, lets you watch agents work from the terminal
2. **Phase 5: Web UI** — richer visualization, shareable view, activity feed
3. **Phase 6: Plugins** — extensibility, auto-approve is the killer feature

Phases 4 and 5 are independent and could be done in either order (or parallel). Phase 6 is independent of both UIs.

The smaller items can be tackled opportunistically between phases.
