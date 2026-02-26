# smoovbrain — Design Document

> AI agent workflow and ticketing system for Claude Code.
> Command: `sb`

## Overview

smoovbrain is an opinionated workflow/ticketing system that sits around Claude Code agents, enforcing process and capturing everything in an Obsidian vault. It combines Claude Code hooks (passive monitoring) and skills (active guidance) to give humans full visibility into what agents are doing, while ensuring agents follow a strict workflow.

The system is designed for **multi-agent** use — multiple Claude sessions can work the board simultaneously, picking up tickets, doing work, and submitting for review.

## Core Principles

- **Files are the database.** Markdown + JSONL. No SQLite, no Postgres, no processes to run.
- **Append-only everything.** MD files grow, JSONL logs grow. History is never rewritten.
- **sb mediates all writes.** Agents and humans interact through the `sb` CLI. No direct file edits to system files.
- **Opinionated workflow.** Fixed stages, enforced transitions, strict rules. Humans can override; agents cannot.
- **Observable by default.** Every hook fires into sb. The JSONL event log is a complete audit trail. TUI and web UI stream it live.

---

## Storage Model

### Location

Two locations with clean separation: Obsidian vault for human-readable markdown, dotdir for machine data.

```
~/.smoovbrain/                           # machine data, config
├── config.toml                          # global config, project registry
└── events/                              # JSONL event logs
    ├── 2026-02-25.jsonl
    └── ...

~/obsidian/smoovbrain/                   # Obsidian vault (default, configurable)
└── tickets/
    ├── 2026-02-25T10:00-sb_a7Kx2m.md   # date prefix for chronological sorting
    ├── 2026-02-25T10:30-sb_b3Yz9q.md
    └── ...
```

The Obsidian vault contains **only markdown** — things humans read. The JSONL event log and config live in `~/.smoovbrain/` — machine data that powers the TUI, web UI, and access control.

### Ticket Files (Markdown)

Each ticket is a single `.md` file with a thin mutable frontmatter and an append-only body.

The frontmatter holds **current state only** — the minimum needed to query/filter tickets without parsing the body. The body is a chronological narrative of everything that has happened to the ticket.

```markdown
---
id: sb_a7Kx2m
title: Add rate limiting to API
project: api-server
status: rework
prior-status: null
assignee: agent-backend-01
priority: P2
depends-on: []
created: 2026-02-25T10:00:00Z
updated: 2026-02-25T14:32:00Z
tags: [api, security]
---

## Created — 2026-02-25T10:00:00Z
**actor:** human

Add rate limiting middleware to all public endpoints. Target: 100 req/min per IP.

## Assigned — 2026-02-25T10:01:00Z
**actor:** sb
**assignee:** agent-backend-01

## In Progress — 2026-02-25T10:02:00Z
**actor:** agent-backend-01 (session: a1b2c3)

Starting work. Found middleware chain in internal/middleware/.

## Review Requested — 2026-02-25T11:30:00Z
**actor:** agent-backend-01 (session: a1b2c3)

Implemented token bucket algorithm in ratelimit.go. Added unit tests.
Coverage: 94% on new code.

## Rejected — 2026-02-25T12:00:00Z
**actor:** agent-review-02 (session: d4e5f6)

No context cancellation handling. Will leak goroutines under load.
Need to wire ctx through the middleware chain.

## Rework — 2026-02-25T12:01:00Z
**actor:** sb
**assignee:** agent-backend-01
```

The body is **never edited**, only appended. When the frontmatter status changes, a new section is appended to the body recording what happened and why.

This means the `.md` file is both a ticket AND its full history, readable as a narrative in Obsidian.

### Event Log (JSONL)

Append-only, machine-readable log of every event in the system. Rotated daily by filename.

```jsonl
{"ts":"2026-02-25T10:00:00Z","event":"ticket.created","ticket":"sb_a7Kx2m","project":"api-server","actor":"human","session":null,"data":{"title":"Add rate limiting to API","priority":"P2"}}
{"ts":"2026-02-25T10:01:00Z","event":"ticket.assigned","ticket":"sb_a7Kx2m","project":"api-server","actor":"sb","session":null,"data":{"assignee":"agent-backend-01"}}
{"ts":"2026-02-25T10:02:00Z","event":"status.in-progress","ticket":"sb_a7Kx2m","project":"api-server","actor":"agent-backend-01","session":"a1b2c3","data":{}}
{"ts":"2026-02-25T10:05:32Z","event":"hook.post-tool","ticket":"sb_a7Kx2m","project":"api-server","actor":"agent-backend-01","session":"a1b2c3","data":{"tool":"Edit","file":"internal/middleware/ratelimit.go"}}
{"ts":"2026-02-25T11:30:00Z","event":"status.review","ticket":"sb_a7Kx2m","project":"api-server","actor":"agent-backend-01","session":"a1b2c3","data":{"note":"Implemented token bucket algorithm"}}
{"ts":"2026-02-25T12:00:00Z","event":"status.rejected","ticket":"sb_a7Kx2m","project":"api-server","actor":"agent-review-02","session":"d4e5f6","data":{"reason":"No context cancellation handling"}}
```

**Two views of the same history:**
- JSONL is optimized for machines — fast parsing, streaming, filtering, access control queries
- MD is optimized for humans — readable in Obsidian, full narrative, wiki-links between tickets

**Daily rotation:** Events go to `events/YYYY-MM-DD.jsonl`. Keeps individual files manageable. Queries that span days read multiple files.

### Why Both?

| Need | JSONL | MD |
|------|-------|----|
| Stream to TUI/web in real-time | tail -f | no |
| Query "who touched this ticket?" | fast scan | slow parse |
| Read the full story of a ticket | painful | beautiful |
| Obsidian graph/linking | no | yes |
| Access control decisions | yes | no |
| Human reviews work | awkward | natural |

They are complementary. sb writes both atomically on every operation.

---

## Workflow

### Statuses

```
BACKLOG → OPEN → IN-PROGRESS → REVIEW ──→ DONE
                      ↑            ↓
                      └── REWORK ←─┘

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
| OPEN | IN-PROGRESS | Must have assignee. Agent picks it up. |
| IN-PROGRESS | REVIEW | Agent must add a note explaining what was done |
| REVIEW | DONE | Reviewer must not have touched the ticket (see Review Rules) |
| REVIEW | REWORK | Reviewer must add rejection reason |
| REWORK | IN-PROGRESS | Assignee picks it back up |
| IN-PROGRESS | REVIEW | (After rework, same rules apply — cycles back through review) |
| (any) | BLOCKED | Requires either a `depends-on` ticket or a human hold reason |
| BLOCKED | (prior) | Auto-unblock when dependency resolves, or human releases hold |

**Hard rules (agents cannot violate these):**
- No skipping stages (no BACKLOG → DONE)
- A ticket in IN-PROGRESS must have exactly one assignee
- An agent cannot review a ticket it has touched in any capacity (enforced via JSONL session history)
- REWORK must go through REVIEW again — no shortcuts to DONE
- Humans can override any rule

### Review Rules

When an agent requests to review a ticket (`sb review sb_xxxxxx`), sb:

1. Scans the JSONL for all events related to the ticket
2. Collects every unique `session` value from those events
3. If the requesting agent's session ID appears → **review denied**

This means an agent that created, was assigned, worked on, or even added a note to a ticket cannot review it. A completely independent agent (or human) must review.

### Dependencies and Blocking

**Ticket-to-ticket dependencies (`depends-on`):**
- A ticket can declare dependencies on other tickets
- If any dependency is not DONE, the ticket is automatically BLOCKED
- When a dependency reaches DONE, sb scans for dependents and auto-unblocks them (snaps back to prior status)
- Creates Obsidian links: `depends-on: [[sb_b3Yz9q]]`

**Human holds:**
- A human can block any ticket with a freeform reason: `sb hold sb_a7Kx2m "waiting on API keys from vendor"`
- Only a human can release a hold: `sb unhold sb_a7Kx2m`

Both are stored as BLOCKED status. The JSONL event records the cause:

```jsonl
{"event":"status.blocked","ticket":"sb_a7Kx2m","reason":"depends-on","ref":"sb_b3Yz9q","prior_status":"OPEN"}
{"event":"status.blocked","ticket":"sb_c1Dw4n","reason":"hold","message":"waiting on API keys","prior_status":"IN-PROGRESS"}
```

The frontmatter stores `prior-status` so the snap-back works:

```yaml
status: blocked
prior-status: open
```

---

## Ticket IDs

Format: `sb_xxxxxx` where `xxxxxx` is 6 characters of **base62** (a-z, A-Z, 0-9).

- 62^6 = ~56.8 billion possible IDs
- Generated randomly, checked for collision against existing tickets
- Short enough to type, long enough to never collide
- Prefix is `sb_` (smoovbrain), hardcoded
- Underscore instead of hyphen — plays nicer with double-click selection and shell word boundaries

Examples: `sb_a7Kx2m`, `sb_Qr9fZw`, `sb_3bNcY1`

---

## Projects

Projects are registered in the global config and correspond to local directories (repos, project folders).

```toml
# ~/.smoovbrain/config.toml

[settings]
vault_path = "~/obsidian/smoovbrain"  # default, configurable

[projects.api-server]
path = "/Users/david/projects/api-server"

[projects.smoovbrain]
path = "/Users/david/projects/smoovbrain"
```

**Registration:** `sb init` in a project directory registers it. Creates a project entry in config.toml.

**Project detection:** When hooks fire or commands run, sb determines the project from `$PWD`. If the current directory is inside a registered project path, events are tagged with that project. If not recognized, events go to an "untracked" bucket (no crash, no noise).

**Cross-project visibility:** Because the vault is user-level, the TUI and web UI can show tickets across all projects. Kanban can filter to one project or show everything.

---

## Agent Session Flow

### Session Start

When a Claude session starts, the `SessionStart` hook fires and calls `sb hook session-start`. sb reads the session context from stdin JSON (which includes `session_id`, `cwd`, `source`), detects the project from `cwd`, and returns an `additionalContext` response with a board summary filtered to the current project:

```
smoovbrain — api-server — 3 OPEN tickets ready

  sb_a7Kx2m  Add rate limiting           P2
  sb_c1Dw4n  Fix CORS headers            P3
  sb_e5Fg8h  Update OpenAPI spec         P4

Pick a ticket with `sb pick sb_xxxxxx`, or use /sb for guided workflow.
To work all 3, spin up an agent team and assign one ticket per teammate.
```

sb has already decided whether to present OPEN or REVIEW work (see Priority Model). The agent gets one clear batch and one job type.

Claude sees this as `additionalContext` injected by the SessionStart hook — it's part of the session context from the very first turn.

### Single Agent Flow

The simplest case: agent picks one ticket and works it.

1. Agent runs `sb pick sb_a7Kx2m` → ticket moves to IN-PROGRESS, assigned to this session
2. Agent does the work, runs `sb note "..."` to document progress
3. Agent runs `sb status review` → ticket moves to REVIEW
4. A different session picks it up for review later

### Batch Work via Agent Teams

When there are multiple OPEN or REVIEW tickets, the agent can use Claude's experimental **agent teams** mode to parallelize work.

**Batch OPEN work:**
```
Agent sees 3 OPEN tickets for api-server, none depend on each other.
Agent spins up a team of 3 worker agents.
Each worker runs `sb pick sb_xxxxxx` on their assigned ticket.
Each worker does their work independently.
Each worker runs `sb status review` when done.
```

**Batch REVIEW work:**
```
Agent sees 2 REVIEW tickets it's eligible to review.
Agent spins up a team of 2 reviewer agents.
Each reviewer runs `sb review sb_xxxxxx` on their assigned ticket.
Each reviewer reads the ticket, checks the work, approves or rejects.
```

**Key rules for team orchestration:**
- The orchestrator agent assigns tickets to teammates — it does NOT pick up tickets itself (staying clean for future reviews)
- OPEN and REVIEW batches should be kept separate (don't mix work types in one team)
- Priority ordering: OPEN tickets worked highest-priority-first
- Each teammate gets its own session ID, so review eligibility is per-session
- The orchestrator can monitor progress via `sb list --project api-server`

### Priority Model

Tickets use a **P0–P5** priority scale:

| Priority | Meaning | Example |
|----------|---------|---------|
| P0 | Critical / outage | Production is down |
| P1 | Urgent | Security vulnerability, data loss risk |
| P2 | High | Core feature work, blocking other tickets |
| P3 | Normal | Standard feature work, most tickets live here |
| P4 | Low | Nice-to-have, polish, minor improvements |
| P5 | Backlog | Someday/maybe, ideas worth tracking |

**How sb decides what to present at session start:**

sb does NOT show both OPEN and REVIEW. It makes the call for the agent:

1. Scan all available tickets for the current project
2. Score them: priority (P0 highest) + status weight (REVIEW gets a small boost since clearing the review queue unblocks others)
3. Pick the top batch of the **same type** — either all OPEN or all REVIEW
4. Present that single batch to the agent

This keeps the agent focused. It gets one job: "work these 3 tickets" or "review these 2 tickets." No decision paralysis.

The orchestrator agent then either works them sequentially or spins up a team to parallelize.

### Orchestrator Disqualification

When an orchestrator agent reads tickets to assign them to teammates, those reads are logged in the JSONL. The orchestrator's session ID appears in the event history for those tickets, which **disqualifies the orchestrator from reviewing them**.

This is intentional. The orchestrator's job is coordination, not review. Review must come from a completely independent session.

---

## CLI: `sb`

Single binary. All commands go through it.

### Core Commands

```
sb init                              # Register current dir as a project
sb new "title" [--priority high] [--depends-on X]  # Create a ticket for current project
sb list [--project X] [--status Y]   # List tickets with filters
sb show sb_xxxxxx                    # Show full ticket detail
sb board                             # TUI kanban (bubbletea)
sb web [--port 8080]                 # Start web UI server
```

### Agent Workflow Commands

```
sb pick [sb_xxxxxx]                  # Pick up a ticket (assigns to current session)
sb note "message"                    # Append a note to current ticket
sb status <status>                   # Transition current ticket to new status
sb review sb_xxxxxx                  # Request to review a specific ticket
sb context                           # Print current session context (used by hooks)
```

### Human Management Commands

```
sb assign sb_xxxxxx <agent-id>       # Manually assign a ticket
sb hold sb_xxxxxx "reason"           # Block a ticket with a human hold
sb unhold sb_xxxxxx                  # Release a human hold
sb override sb_xxxxxx <status>       # Force a status (human override)
sb close sb_xxxxxx                   # Mark done (human shortcut)
```

### Hook Commands

```
sb hook session-start                # Called by SessionStart — returns board context
sb hook pre-tool                     # Called by PreToolUse — logs tool call
sb hook post-tool                    # Called by PostToolUse — logs tool result
sb hook subagent-start               # Called by SubagentStart — injects ticket context
sb hook subagent-stop                # Called by SubagentStop — logs completion
sb hook task-completed               # Called by TaskCompleted — logs completion
sb hook teammate-idle                # Called by TeammateIdle — logs idle state
sb hook permission-request           # Called by PermissionRequest — auto-approve (plugin)
sb hook stop                         # Called by Stop — logs session stop
sb hook session-end                  # Called by SessionEnd — cleanup
```

All hook commands read JSON from stdin (common fields: `session_id`, `cwd`, `hook_event_name`, plus event-specific fields). They write to the JSONL event log and optionally return JSON on stdout for context injection or decision control.

### Ergonomics

Commands should be forgiving:
- `sb status review`, `sb review`, `sb submit` → all do the same thing
- `sb status in-progress`, `sb start`, `sb begin` → all do the same thing
- Clear error messages with guidance: "Can't move to REVIEW — no notes since starting work. Run `sb note` first."
- `sb help` is context-aware: shows relevant commands based on current ticket status

---

## Claude Code Integration

### Hook Protocol

All hooks receive JSON on stdin with common fields: `session_id`, `cwd`, `transcript_path`, `permission_mode`, `hook_event_name`, plus event-specific fields. sb reads stdin, extracts context, and writes to the JSONL event log.

### Hooks

Global hooks in `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [{ "type": "command", "command": "sb hook session-start" }]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "*",
        "hooks": [{ "type": "command", "command": "sb hook pre-tool", "async": true }]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "*",
        "hooks": [{ "type": "command", "command": "sb hook post-tool", "async": true }]
      }
    ],
    "SubagentStart": [
      {
        "hooks": [{ "type": "command", "command": "sb hook subagent-start" }]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [{ "type": "command", "command": "sb hook subagent-stop", "async": true }]
      }
    ],
    "TaskCompleted": [
      {
        "hooks": [{ "type": "command", "command": "sb hook task-completed", "async": true }]
      }
    ],
    "TeammateIdle": [
      {
        "hooks": [{ "type": "command", "command": "sb hook teammate-idle", "async": true }]
      }
    ],
    "Stop": [
      {
        "hooks": [{ "type": "command", "command": "sb hook stop", "async": true }]
      }
    ],
    "PermissionRequest": [
      {
        "hooks": [{ "type": "command", "command": "sb hook permission-request" }]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [{ "type": "command", "command": "sb hook session-end", "async": true }]
      }
    ]
  }
}
```

### Hook Behaviors

| Hook | Blocking? | What sb does |
|------|-----------|-------------|
| `session-start` | **Yes** (returns context) | Detects project from `cwd`. Returns `additionalContext` with board summary. Injects sb workflow instructions into the session. |
| `pre-tool` | No (async) | Logs tool call to JSONL. |
| `post-tool` | No (async) | Logs tool result to JSONL. |
| `subagent-start` | **Yes** (returns context) | Logs subagent spawn. Returns `additionalContext` to inject sb workflow instructions into the subagent (which ticket to work on, commands to use). |
| `subagent-stop` | No (async) | Logs subagent completion. |
| `task-completed` | No (async) | Logs task completion to JSONL. Does not block — ticket completion is decided by reviewers via `sb review`, not by the task system. |
| `teammate-idle` | No (async) | Logs teammate idle state for monitoring. |
| `permission-request` | **Yes** (can auto-approve) | If auto-approve plugin is enabled and agent has an active ticket, auto-approves trusted tools. Otherwise passes through. |
| `stop` | No (async) | Logs session stop. |
| `session-end` | No (async) | Logs session end. Cleans up any session state. |

**Key design choice:** Most hooks are `async: true` (non-blocking). Only `session-start` and `subagent-start` are synchronous because they need to inject context. Ticket completion is enforced by reviewers via `sb review`, not by hooks. This keeps sb from slowing down the agent.

### SubagentStart Context Injection

This is how sb gets workflow instructions into subagents spawned by an orchestrator. When an agent team is spun up:

1. Orchestrator spawns a teammate via Claude Code's Task tool
2. `SubagentStart` hook fires → `sb hook subagent-start` runs
3. sb reads the subagent's context, checks if the orchestrator has assigned it a ticket
4. sb returns `additionalContext` with the ticket details and workflow commands

This means teammates automatically know which ticket they're working on and what sb commands to use — the orchestrator doesn't have to explain the sb workflow in every task prompt.

---

## Web UI

### Tech Stack

- **Go** net/http server, embedded in the `sb` binary
- **templ** for type-safe Go HTML templates
- **htmx** for dynamic interactions without a JS framework
- **Franken UI** (UIKit + Tailwind) for styling
- **SSE (Server-Sent Events)** via htmx extension for real-time updates

### Serving

```
sb web [--port 8080]
```

Static assets (CSS, JS, htmx, Franken UI) are embedded in the binary via `go:embed`. No separate build step, no node_modules.

### Views

**Kanban board** — columns for each status, cards show ticket ID, title, project, assignee. Cards move between columns in real-time via SSE as events come in.

**List view** — tickets grouped by project, sortable/filterable by status, priority, assignee, age.

**Ticket detail** — full rendered markdown of the ticket file. Shows the complete narrative. Links to dependent/blocking tickets.

**Activity feed** — live stream of JSONL events. Filterable by project, agent, ticket.

### Real-time Updates

The server tails the JSONL event log files and pushes updates via SSE. The htmx SSE extension on the client receives events and swaps in updated HTML fragments. No WebSocket complexity, no JS state management.

---

## TUI

### Tech Stack

- **bubbletea** for the terminal UI framework
- **lipgloss** for styling
- **bubbles** for common components (tables, text inputs, viewports)

### Views

**List view (default):**
```
┌─ api-server ─────────────────────────────────────────────┐
│ sb_a7Kx2m  Add rate limiting       IN-PROGRESS  P2  ●●● │
│ sb_c1Dw4n  Fix CORS headers        REVIEW       P3  ●●○ │
│ sb_e5Fg8h  Update OpenAPI spec     BACKLOG      P4  ●○○ │
├─ smoovbrain ─────────────────────────────────────────────┤
│ sb_b3Yz9q  Implement TUI board     IN-PROGRESS  P2  ●●● │
│ sb_d2Ex7p  Add SSE endpoint        OPEN         P3  ●●○ │
└──────────────────────────────────────────────────────────┘
```

**Kanban view:**
```
│ BACKLOG    │ OPEN       │ IN-PROGRESS │ REVIEW     │ DONE       │
│            │            │             │            │            │
│ sb_e5Fg8h │ sb_d2Ex7p  │ sb_a7Kx2m  │ sb_c1Dw4n │            │
│ Update Op… │ Add SSE e… │ Add rate l… │ Fix CORS … │            │
│ api-server │ smoovbrain │ api-server  │ api-server │            │
│            │            │             │            │            │
│            │            │ sb_b3Yz9q  │            │            │
│            │            │ Implement … │            │            │
│            │            │ smoovbrain  │            │            │
```

Toggle between views with a keypress. Filter by project, status, priority.

---

## Architecture (Go Packages)

```
smoovbrain/
├── cmd/
│   └── sb/
│       └── main.go              # CLI entrypoint (cobra)
├── internal/
│   ├── config/                  # Config loading (TOML)
│   ├── ticket/                  # Ticket CRUD, MD read/write
│   ├── event/                   # JSONL event log, append, query, tail
│   ├── workflow/                # State machine, transition rules, validation
│   ├── project/                 # Project detection from PWD
│   ├── identity/                # Session ID resolution
│   ├── hook/                    # Hook command handlers
│   ├── plugin/                  # Plugin loading, event fan-out
│   ├── tui/                     # bubbletea TUI
│   │   ├── board/               # Kanban view
│   │   └── list/                # List view
│   └── web/                     # Web server
│       ├── handler/             # HTTP handlers
│       ├── sse/                 # SSE event streaming
│       ├── templates/           # templ templates
│       └── static/              # Embedded static assets
├── DESIGN.md                    # This file
├── go.mod
└── go.sum
```

---

## Plugin System

sb has an internal plugin system that lets hook events trigger additional behaviors. Plugins are configured in `config.toml` and receive events from the JSONL stream. This is NOT a Claude Code plugin — it's sb's own extension mechanism.

### Why Plugins

The core hook handlers (log to JSONL, inject context, enforce workflow) are built into sb. But there are behaviors that are user-specific, environment-specific, or experimental — things that don't belong in the core but should be easy to wire up:

- **Auto-approval:** Automatically approve Claude Code permission prompts for trusted operations during sb-managed work
- **Audio alerts:** Play a sound when a ticket changes state, a review is needed, or an agent gets stuck
- **Notifications:** Desktop notifications, Slack webhooks, email on specific events
- **Metrics:** Track cycle time, throughput, agent efficiency
- **Custom validation:** Project-specific rules for review acceptance or completion criteria

### Plugin Configuration

```toml
# ~/.smoovbrain/config.toml

[[plugins]]
name = "audio-alerts"
enabled = true
command = "sb-plugin-audio"      # executable on PATH, or absolute path
events = ["status.*", "ticket.created"]

[plugins.config]
player = "afplay"                # "afplay" (macOS), "paplay" (Linux/NixOS)
tts_command = "say"              # "say" (macOS), "espeak" (Linux/NixOS)
speak_events = true              # TTS announces ticket transitions
sound_review = "glass"           # macOS system sound
sound_done = "hero"
sound_blocked = "basso"

[[plugins]]
name = "auto-approve"
enabled = true
command = "sb-plugin-approve"
events = ["hook.permission-request"]

[plugins.config]
# Auto-approve Bash when working on an sb ticket (other tools are safe by default)
allow_bash = true
# Only when the agent has an active sb ticket assigned
require_active_ticket = true

[[plugins]]
name = "slack-notify"
enabled = false
command = "sb-plugin-slack"
events = ["status.review", "status.done", "status.blocked"]

[plugins.config]
webhook_url = "https://hooks.slack.com/services/xxx/yyy/zzz"
channel = "#dev-agents"
```

### Plugin Protocol

Plugins are external executables that receive events on stdin as JSON (same format as JSONL events) and can optionally return JSON on stdout.

```
sb event occurs → sb writes to JSONL → sb fans out to matching plugins

Plugin stdin:  {"ts":"...","event":"status.review","ticket":"sb_a7Kx2m",...}
Plugin stdout: (optional JSON response, plugin-specific)
Plugin stderr: (logged by sb for debugging)
```

**Event matching:** The `events` field in config uses glob patterns against event names. `"status.*"` matches `status.review`, `status.blocked`, etc. `"*"` matches everything.

**Lifecycle:** Plugins are invoked per-event (short-lived process), not long-running daemons. This keeps things simple — no socket management, no health checks. For plugins that need to batch events (like metrics aggregation), they can maintain their own state files.

### Built-in Plugin: Auto-Approve (Future)

The auto-approve plugin would hook into Claude Code's `PermissionRequest` hook (not sb's plugin system directly — it would need to be wired as a Claude Code hook). When an agent has an active sb ticket, the plugin auto-approves trusted tool calls:

1. `PermissionRequest` hook fires → `sb hook permission-request` runs
2. sb checks: does this session have an active ticket?
3. sb checks: is the tool in the allow list?
4. If yes to both → return `hookSpecificOutput` with `decision.behavior: "allow"`

This is powerful but dangerous — it should be opt-in, configurable per-project, and logged extensively.

### Built-in Plugin: Audio Alerts (Future)

Uses configurable player and TTS commands. Platform-specific settings live in `config.toml`.

```bash
# Conceptual implementation
#!/bin/bash
INPUT=$(cat /dev/stdin)
EVENT=$(echo "$INPUT" | jq -r '.event')
TICKET=$(echo "$INPUT" | jq -r '.ticket')
TITLE=$(echo "$INPUT" | jq -r '.data.title // empty')

# Sound effect
case "$EVENT" in
  status.review)   $PLAYER /System/Library/Sounds/Glass.aiff ;;
  status.done)     $PLAYER /System/Library/Sounds/Hero.aiff ;;
  status.blocked)  $PLAYER /System/Library/Sounds/Basso.aiff ;;
esac

# TTS announcement
if [ "$SPEAK_EVENTS" = "true" ]; then
  case "$EVENT" in
    status.review)   $TTS_COMMAND "Ticket $TICKET ready for review" ;;
    status.done)     $TTS_COMMAND "Ticket $TICKET completed" ;;
    status.blocked)  $TTS_COMMAND "Ticket $TICKET is blocked" ;;
  esac
fi
```

These ship as example scripts in the repo, not compiled into the binary.

---

## Claude Code Task System Compatibility

Claude Code has a built-in task system (TaskCreate, TaskUpdate, TaskList, TaskGet) used for coordinating work within agent teams. smoovbrain tickets and Claude Code tasks are **different layers** that work together:

| | Claude Code Tasks | smoovbrain Tickets |
|--|--|--|
| **Scope** | Single agent team session | Cross-session, persistent |
| **Lifetime** | Dies when session ends | Lives until DONE |
| **Purpose** | Coordinate teammates within one team | Track work across days/weeks/projects |
| **Created by** | Team lead via TaskCreate | Human or agent via `sb new` |
| **Visible to** | Teammates in that session | Everyone, forever (Obsidian + web UI) |

### How They Interact

When an orchestrator agent picks up a batch of smoovbrain tickets and creates a team:

1. **Orchestrator reads sb tickets** → `sb list --project X --status open`
2. **Orchestrator creates Claude Code tasks** for each teammate → uses TaskCreate to assign "Work on sb_a7Kx2m"
3. **Each teammate starts** → `SubagentStart` hook fires, sb injects ticket context
4. **Teammate runs `sb pick sb_a7Kx2m`** → sb moves ticket to IN-PROGRESS
5. **Teammate does the work** → PreToolUse/PostToolUse hooks log activity to JSONL
6. **Teammate runs `sb status review`** → sb moves ticket to REVIEW
7. **Teammate marks Claude Code task complete** → `TaskCompleted` hook fires, sb logs it to JSONL
8. **Session ends** → Claude Code tasks vanish, but sb tickets persist with full history

The `TaskCompleted` hook logs the event but does not block. Ticket completion is decided by reviewers via `sb review`, not by the task system. The two systems have different lifecycles and sb does not interfere with Claude Code's task management.

### What sb Does NOT Do

- sb does NOT replace Claude Code's task system — it sits above it
- sb does NOT create Claude Code tasks — the orchestrator agent does that natively
- sb does NOT try to sync state between the two systems — they have different lifecycles
- Agents use Claude Code tasks for intra-session coordination and sb tickets for persistent workflow

---

## Resolved Questions

- [x] **Session ID sourcing:** Comes via stdin JSON as `session_id` field in the common input. All hooks get it.
- [x] **Hook input format:** JSON on stdin with common fields (`session_id`, `cwd`, `transcript_path`, `permission_mode`, `hook_event_name`) plus event-specific fields. Documented in the Hook Protocol section.
- [x] **Session start hook:** `SessionStart` hook exists, fires on `startup`/`resume`/`clear`/`compact`. Can inject `additionalContext` and persist env vars via `CLAUDE_ENV_FILE`.
- [x] **Agent teams hooks:** `SubagentStart`, `SubagentStop`, `TeammateIdle`, `TaskCompleted` all available. SubagentStart can inject context into subagents.
- [x] **Priority system:** P0–P5 scale.
- [x] **PLANNED → OPEN:** Renamed.
- [x] **Batch type selection:** sb decides (not the agent). Presents one batch type based on priority scoring.
- [x] **Orchestrator review eligibility:** Disqualified. Reads are logged, session ID appears in history.
- [x] **Orchestrator ticket assignment to subagents:** Option (b) — orchestrator includes ticket ID in the Task prompt, `SubagentStart` hook parses it and injects full ticket context via `additionalContext`. Simple, no pre-registration needed. Subagents don't get `SessionStart` — only `SubagentStart`, which is the right seam for this.
- [x] **sb new priority default:** Defaults to P3 (Normal) when no priority specified.
- [x] **Plugin execution model:** Per-event invocation (stateless). MD and JSONL files are the state — no daemon, no long-running process. `sb` is a pure CLI: read files, do work, write files, exit. Only `sb web` and `sb board` are long-lived (optional UIs that tail files, own no state).

- [x] **Skills:** Removed. SessionStart + SubagentStart hooks inject enough context. No separate skill layer needed — can add later if hooks prove insufficient.
- [x] **Ticket file names:** Flat directory with date-prefixed filenames: `2026-02-25T10:00-sb_xxxxxx.md`. Creation timestamp gives chronological sorting in Obsidian and `ls`. Ticket ID is still the canonical identifier.
- [x] **Config format:** TOML. `BurntSushi/toml` in Go.
- [x] **Event log retention:** Keep JSONL forever for v1. Daily rotation keeps files manageable. Add `sb gc` later if needed.
- [x] **Review batch eligibility filtering:** Scan JSONL on session start for v1. Daily files keep it fast. Add index cache later if needed.
- [x] **Auto-approve safety:** Focused on Bash (the only dangerous tool). Other tools (Edit, Write, Read, etc.) are safe by default. `require_active_ticket` guard ensures agent is on an sb ticket.
- [x] **Audio alerts cross-platform:** Config-driven via `player` and `tts_command` in TOML. macOS: `afplay`/`say`. Linux/NixOS: `paplay`/`espeak`. TTS announces ticket transitions ("Ticket sb_a7Kx2m ready for review"). Example scripts ship in repo.

## Out of Spec

- **Multi-machine sync:** Not designing for it. Single machine only.

## Future Plugin Ideas

- **Observability:** Webhook plugin that POSTs event payloads to configurable endpoints. Enables integration with external dashboards, alerting systems, or custom tooling without modifying sb core.
