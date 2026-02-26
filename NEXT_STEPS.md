# smoovtask — Next Steps

## What's Done (Phases 1–4)

The core CLI and web UI are complete.

### Commands (17 total)
- `init`, `new`, `list`, `show`, `pick`, `note`, `status`, `review`
- `assign`, `hold`, `unhold`, `close`, `override`, `context`
- `hook <event>` (10 handlers), `hooks install`
- `web` — browser-based dashboard

### Internals
- Workflow state machine with transition rules, review eligibility, dependency auto-blocking/unblocking
- Markdown tickets with YAML frontmatter in Obsidian vault
- JSONL event log with daily rotation and flock-based append
- Project detection from PWD, session identity from env vars
- Priority-scored batch selection in session-start hook

### Phase 4: Web UI — `st web` (done)

Browser dashboard with live updates. See [PHASE_4.md](PHASE_4.md) for full details.

- **Kanban board** (`/`) — tickets grouped by status, live SSE updates
- **List view** (`/list`) — filterable table by project and status
- **Ticket detail** (`/ticket/{id}`) — goldmark-rendered markdown body + metadata sidebar
- **Activity feed** (`/activity`) — recent events with project/type filters, live updates
- **SSE streaming** (`/events`) — fsnotify watches JSONL dir, fan-out broker pushes to clients
- **Tech**: templ templates, htmx + SSE extension, Franken UI (dark theme), goldmark, vendored via go:embed

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
| `internal/web/handler/` | 13 tests (handlers, SSE broker, watcher) |

---

## What's Left

### Phase 5: TUI — `st board`

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

1. **Phase 5: TUI** — immediate value, lets you watch agents work from the terminal
2. **Phase 6: Plugins** — extensibility, auto-approve is the killer feature

Phases 5 and 6 are independent and could be done in either order (or parallel).

The smaller items can be tackled opportunistically between phases.
