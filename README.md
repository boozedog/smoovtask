# smoovtask

AI agent workflow and ticketing system for [Claude Code](https://docs.anthropic.com/en/docs/claude-code), OpenCode, and PI. Command: `st`

An opinionated workflow/ticketing system that sits around Claude Code agents, enforcing process and capturing everything in an Obsidian vault. Multiple Claude sessions can work the board simultaneously — picking up tickets, doing work, and submitting for review.

## Quickstart

```bash
# 1. Install
go install github.com/boozedog/smoovtask/cmd/st@latest

# 2. Register a project
cd ~/projects/my-app
st init

# 3. Install agent hooks/extensions
st hooks install --agents both
# or: st hooks install --agents pi

# 4. Create a ticket
st new "Add rate limiting to API" --priority P2

# 5. Start an agent session — st injects the board automatically
# Claude sees open tickets and picks one up:
#   st pick st_a7Kx2m
#   st note "Implemented token bucket middleware"
#   st status review
```

That's it. Agent sessions now see the ticket board on startup, and all activity is logged.

## Workflow

### Statuses

```
BACKLOG → OPEN → IN-PROGRESS → REVIEW ──→ DONE
   ↑          ↑        ↑            ↓        │
   │          │        └── REWORK ←─┘        │
   └──────────┴────────────────────────────── ┘
                  (any status → BACKLOG)

      BLOCKED ←── (any status)
      BLOCKED ──→ (snaps back to prior status)
```

| Status | Meaning |
|--------|---------|
| BACKLOG | Identified but not yet scoped |
| OPEN | Scoped and ready to be picked up |
| IN-PROGRESS | Actively being worked on by an agent |
| REVIEW | Submitted for review, awaiting a different agent or human |
| REWORK | Review rejected, needs changes |
| DONE | Completed and accepted |
| BLOCKED | Cannot proceed — depends on another ticket or human hold |

### Transition Rules

| From | To | Conditions |
|------|-----|-----------|
| BACKLOG | OPEN | Human or agent |
| OPEN | IN-PROGRESS | Must have assignee (agent picks it up) |
| IN-PROGRESS | REVIEW | Agent submits for review |
| REVIEW | DONE | Reviewer must not have touched the ticket |
| REVIEW | REWORK | Reviewer adds rejection reason |
| REWORK | IN-PROGRESS | Assignee picks it back up |
| (any) | BACKLOG | Deprioritize — clears assignee, no snap-back |
| (any) | BLOCKED | Requires a dependency or human hold reason |
| BLOCKED | (prior) | Auto-unblocks when dependency resolves or human releases hold |

**Hard rules (agents cannot violate):**

- No skipping stages (no BACKLOG → DONE)
- A ticket in IN-PROGRESS must have exactly one assignee
- An agent cannot review a ticket it has touched in any capacity
- REWORK must go through REVIEW again — no shortcuts to DONE
- Humans can override any rule via `st override`

### Review Eligibility

When an agent requests review (`st review st_xxxxxx`), smoovtask scans the JSONL event log for all sessions that have touched the ticket. If the requesting agent's session ID appears anywhere in that history, the review is denied. A completely independent session (or human) must review.

### Dependencies and Blocking

**Ticket-to-ticket dependencies:** A ticket can declare `--depends-on` at creation. If any dependency is not DONE, the ticket is automatically BLOCKED. When a dependency reaches DONE, smoovtask auto-unblocks dependents (snaps back to prior status).

**Human holds:** Block any ticket with a freeform reason (`st hold`). Only a human can release it (`st unhold`).

### Priority

Tickets use a P0–P5 scale. Default is P3.

| Priority | Meaning |
|----------|---------|
| P0 | Critical / outage |
| P1 | Urgent |
| P2 | High |
| P3 | Normal (default) |
| P4 | Low |
| P5 | Backlog / someday |

At session start, smoovtask scores available tickets by priority and status weight, then presents a single batch — either all OPEN or all REVIEW — to keep the agent focused.

## Usage

### Core

```
st init                                    Register current directory as a project
st new <title> [--priority P0-P5]          Create a ticket (default: P3)
       [--title T]                         Title as flag (alternative to positional)
       [--tags a,b]
       [--depends-on st_x,st_y]
st list [--project X] [--status Y]         List tickets (auto-detects project from PWD)
st show <ticket-id>                        Show full ticket detail (frontmatter + body)
```

### Agent Workflow

```
st pick [ticket-id]                        Pick up a ticket (assigns to current session)
                                           Omit ID to auto-pick highest-priority open ticket
st note <message>                          Append a note to the current ticket
st status <status>                         Transition ticket status
                                           Aliases: review/submit, start/begin, done/complete
st review <ticket-id>                      Claim a ticket for review (eligibility enforced)
st context                                 Print current session context as JSON
```

### Human Management

```
st assign <ticket-id> <agent-id>           Manually assign a ticket
st hold <ticket-id> <reason>               Block a ticket with a human hold
st unhold <ticket-id>                      Release a human hold
st override <ticket-id> <status>           Force-set status (bypasses all rules)
st close <ticket-id>                       Mark done (human shortcut, bypasses workflow)
```

### Hooks

```
st hooks install [--agents ...]            Install Claude hooks and optional OpenCode/PI bridges
st hook <event-type>                       Handle a Claude Code hook event (10 handlers)
```

For OpenCode/PI integration, use:

```bash
st hooks install --agents opencode
st hooks install --agents pi
```

Hook event types: `session-start`, `pre-tool`, `post-tool`, `subagent-start`, `subagent-stop`, `task-completed`, `teammate-idle`, `permission-request`, `stop`, `session-end`

## Agent Integrations

smoovtask integrates with Claude Code through [hooks](https://docs.anthropic.com/en/docs/claude-code/hooks), and with OpenCode/PI through installed TypeScript plugins/extensions that forward lifecycle events to `st hook`.

### Installing Hooks/Extensions

```bash
st hooks install
# optional: st hooks install --agents opencode
# optional: st hooks install --agents pi
```

This adds smoovtask hooks to `~/.claude/settings.json`, and installs OpenCode/PI bridge files when requested, preserving existing settings.

### What Each Hook Does

| Hook | Blocking | Behavior |
|------|----------|----------|
| `session-start` | Yes | Detects project from `cwd`, returns board summary as `additionalContext` |
| `pre-tool` | No | Logs tool call to JSONL event log |
| `post-tool` | No | Logs tool result to JSONL event log |
| `subagent-start` | Yes | Injects ticket context into subagents via `additionalContext` |
| `subagent-stop` | No | Logs subagent completion |
| `task-completed` | No | Logs task completion (does not affect ticket status) |
| `teammate-idle` | No | Logs teammate idle state for monitoring |
| `permission-request` | Yes | Can auto-approve trusted tools when agent has active ticket |
| `stop` | No | Logs session stop |
| `session-end` | No | Logs session end, cleans up session state |

Only `session-start`, `subagent-start`, and `permission-request` are synchronous (blocking). Everything else runs async to avoid slowing down the agent.

### Session Start Context

When a Claude Code session starts, the `session-start` hook injects a board summary showing available tickets for the current project:

```
smoovtask — api-server — 3 OPEN tickets ready

  st_a7Kx2m  Add rate limiting           P2
  st_c1Dw4n  Fix CORS headers            P3
  st_e5Fg8h  Update OpenAPI spec         P4

Pick a ticket with `st pick st_xxxxxx`.
```

### Subagent Context

When an orchestrator spins up agent teammates, the `subagent-start` hook automatically injects the assigned ticket's details and workflow commands into each subagent — the orchestrator doesn't need to explain the smoovtask workflow in every task prompt.

### Multi-Agent Work

smoovtask is designed for multiple Claude sessions working simultaneously:

1. Orchestrator reads the board via `st list`
2. Spins up a team of worker agents, assigning one ticket each
3. Each worker runs `st pick`, does the work, runs `st status review`
4. A separate session reviews completed tickets via `st review`

The orchestrator's session ID is logged when it reads tickets, which disqualifies it from reviewing those tickets — ensuring independent review.

## Architecture

### Package Structure

```
smoovtask/
├── cmd/st/main.go              CLI entrypoint
├── cmd/                        Cobra commands (one file per command)
├── internal/
│   ├── config/                 TOML config loading, project registry
│   ├── ticket/                 Ticket struct, ID gen, markdown parse/write, file store
│   ├── event/                  JSONL event log: append (flock), daily rotation, query
│   ├── workflow/               State machine, transition rules, review eligibility
│   ├── project/                Project detection from PWD
│   ├── identity/               Invocation identity (`--run-id` / `--human`)
│   └── hook/                   Hook command handlers (10 event types)
├── DESIGN.md                   Full design document
├── NEXT_STEPS.md               Roadmap and future phases
├── justfile                    Build/test/lint commands
├── go.mod
└── go.sum
```

### Storage Model

Two locations with clean separation:

```
~/.smoovtask/                           Machine data + config
├── config.toml                          Global config, project registry
└── events/                              JSONL event logs (daily rotation)
    └── YYYY-MM-DD.jsonl

~/obsidian/smoovtask/                   Obsidian vault (configurable)
└── tickets/
    └── YYYY-MM-DDTHH:MM-st_xxxxxx.md   Markdown tickets
```

### Ticket Format

Each ticket is a markdown file with YAML frontmatter and an append-only body:

```markdown
---
id: st_a7Kx2m
title: Add rate limiting to API
project: api-server
status: in-progress
assignee: agent-backend-01
priority: P2
depends-on: []
created: 2026-02-25T10:00:00Z
updated: 2026-02-25T10:02:00Z
tags: [api, security]
---

## Created — 2026-02-25T10:00:00Z
**actor:** human

Add rate limiting middleware to all public endpoints.

## In Progress — 2026-02-25T10:02:00Z
**actor:** agent-backend-01 (session: a1b2c3)

Starting work. Found middleware chain in internal/middleware/.
```

The frontmatter holds current state. The body is a chronological narrative — never edited, only appended. Each ticket is both a ticket and its full history, readable in Obsidian.

### Event Log

Append-only JSONL, rotated daily:

```jsonl
{"ts":"...","event":"ticket.created","ticket":"st_a7Kx2m","project":"api-server","actor":"human","data":{"title":"Add rate limiting","priority":"P2"}}
{"ts":"...","event":"status.in-progress","ticket":"st_a7Kx2m","project":"api-server","actor":"agent-backend-01","session":"a1b2c3","data":{}}
```

### Configuration

```toml
# ~/.smoovtask/config.toml

[settings]
vault_path = "~/obsidian/smoovtask"

[projects.api-server]
path = "/Users/david/projects/api-server"

[projects.smoovtask]
path = "/Users/david/projects/smoovtask"
```

## Development

### Prerequisites

- Go 1.25+
- [just](https://github.com/casey/just) (command runner)
- [gofumpt](https://github.com/mvdan/gofumpt) (formatter)
- [golangci-lint](https://golangci-lint.run/) (linter)

### Commands

```bash
just build          # Build + install to GOPATH/bin
just test           # go test -v ./...
just test-short     # go test ./... (non-verbose)
just test-cover     # Tests with HTML coverage report
just lint           # golangci-lint
just fmt            # gofumpt (not gofmt)
just vuln           # govulncheck
just release        # goreleaser snapshot build
just clean          # Remove build artifacts
```

### Formatting

This project uses **gofumpt** (stricter than gofmt). Always run `just fmt` before committing. Tabs for indentation (enforced).

## Roadmap

The core CLI is complete. Future phases (see [NEXT_STEPS.md](NEXT_STEPS.md)):

- **TUI** (`st board`) — Terminal kanban board using bubbletea + lipgloss
- **Web UI** (`st web`) — Embedded web server with htmx + SSE for real-time updates
- **Plugin system** — Event-driven extensions (auto-approve, audio alerts, Slack notifications)
