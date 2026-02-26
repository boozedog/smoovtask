# smoovtask Phase 2: Complete Workflow + Hook Installation

## Context for the Next Agent

Phase 1 is complete and pushed. You have a working `st` CLI with:

- **Scaffold**: Go 1.26, Cobra, golangci-lint, gofumpt, goreleaser, justfile, hk.pkl, GitHub rulesets
- **Config**: `~/.smoovtask/config.toml` with project registry, vault path, tilde expansion
- **Ticket CRUD**: Create/Get/List/Save with markdown files (YAML frontmatter + append-only body)
- **Event Log**: JSONL with flock-protected append, daily rotation, query/filter, `SessionsForTicket()`
- **Workflow Engine**: State machine (BACKLOG→OPEN→IN-PROGRESS→REVIEW→DONE/REWORK), status aliases, review eligibility
- **Hook Handlers**: session-start (board summary), subagent-start (ticket context injection), pre/post-tool logging, stop logging
- **CLI Commands**: `init`, `new`, `list`, `show`, `pick`, `note`, `status`, `review`, `hook`

All tests pass (`go test ./...`), lint is clean (`golangci-lint run ./...`).

Read `DESIGN.md` for the full spec — it's the source of truth for all behavior.

## What Phase 2 Covers

### Step 1: Install Claude Code Hooks

Create or update `~/.claude/settings.json` to wire all `st hook` commands. The hook config is fully specified in DESIGN.md under "Hooks". Add an `st hooks install` command that merges the hook config into the user's existing settings.json (don't clobber existing settings).

**Files:**
- `cmd/hooks_install.go` — `st hooks install` command

**Key details:**
- Read existing `~/.claude/settings.json`, merge smoovtask hooks into the `hooks` key
- SessionStart, PreToolUse, PostToolUse are already implemented
- SubagentStart, Stop are already implemented
- Mark unimplemented hooks with comments (subagent-stop, task-completed, teammate-idle, permission-request, session-end)
- Async hooks: pre-tool, post-tool, subagent-stop, task-completed, teammate-idle, stop, session-end
- Sync hooks: session-start, subagent-start, permission-request

### Step 2: Missing Workflow Commands

**Files:**
- `cmd/hold.go` — `st hold st_xxxxxx "reason"` — block ticket with human hold
- `cmd/unhold.go` — `st unhold st_xxxxxx` — release human hold
- `cmd/assign.go` — `st assign st_xxxxxx <agent-id>` — manually assign
- `cmd/close.go` — `st close st_xxxxxx` — human shortcut to mark done
- Update `cmd/new.go` — add `--depends-on` flag (replaces `st spawn`)

Each command must:
1. Validate the operation (check current status, permissions)
2. Update the ticket markdown (frontmatter + append section)
3. Log a JSONL event

### Step 3: Dependency and Blocking System

**Files:**
- `internal/ticket/deps.go` + test — dependency resolution, auto-block/unblock logic
- Update `internal/workflow/machine.go` — BLOCKED transitions use `prior-status` snap-back

**Key details from DESIGN.md:**
- `depends-on` field in frontmatter lists ticket IDs
- If any dependency is not DONE, ticket auto-transitions to BLOCKED with `prior-status` saved
- When a dependency reaches DONE, scan for dependents and auto-unblock (snap back to prior-status)
- Human holds: `st hold` sets BLOCKED with reason in JSONL, only `st unhold` releases
- Frontmatter stores `prior-status` so snap-back works
- Two kinds of BLOCKED: `depends-on` (auto) and `hold` (manual)

Wire auto-unblock into `cmd/status.go`: when a ticket moves to DONE, scan all tickets for dependents that can now unblock.

### Step 4: Priority Sorting + Batch Selection

**Files:**
- Update `internal/hook/session_start.go` — sort tickets by priority (P0 first), select review batch over open batch with proper scoring

**Key details from DESIGN.md:**
- Score: priority weight + status weight (REVIEW gets small boost)
- Present one batch type: all OPEN or all REVIEW, not mixed
- Sort by priority within the batch (P0 first)

### Step 5: Remaining Hook Handlers

**Files:**
- `internal/hook/subagent_stop.go` — log subagent completion
- `internal/hook/task_completed.go` + test — log task completion to JSONL (no blocking)
- `internal/hook/teammate_idle.go` — log idle state
- `internal/hook/permission_request.go` — pass-through for now (plugin system handles auto-approve later)
- `internal/hook/session_end.go` — cleanup logging
- Update `cmd/hook.go` — add cases for new event types

**task-completed key behavior:** Log-only. Does not block. Ticket completion is decided by reviewers via `st review`, not by the task system.

### Step 6: Improve `st status` Current-Ticket Resolution

Currently `resolveCurrentTicket` in `cmd/status.go` scans tickets by assignee. Improve to also work when `--ticket` is not set and there's no `CLAUDE_SESSION_ID` — prompt the user or show guidance.

Also: make `st note` not depend on the `statusTicket` package var hack — extract `resolveCurrentTicket` into a shared function in `cmd/helpers.go`.

## Parallelization

- **Steps 1 + 2** can run in parallel (independent)
- **Step 3** depends on Step 2 (needs hold/unhold commands)
- **Step 4** is independent
- **Step 5** is independent (but test after Step 1 installs hooks)
- **Step 6** is a small refactor, can run anytime

## Verification

After all steps:
1. `st hooks install` adds hooks to `~/.claude/settings.json`
2. `st new "Dep test A"` → `st new "Dep test B" --depends-on st_xxxxxx` (no separate spawn command)
3. Dep test B should auto-BLOCK
4. Move Dep test A through to DONE → Dep test B auto-unblocks
5. `st hold st_yyyy "waiting on keys"` → ticket BLOCKED
6. `st unhold st_yyyy` → ticket snaps back
7. `st assign st_yyyy agent-99` works
8. `st close st_yyyy` marks DONE
9. `echo '...' | st hook task-completed` — logs event to JSONL (no blocking)
10. Priority sorting: P0 tickets appear first in session-start board
11. `go test ./...` all pass
12. `golangci-lint run ./...` clean

## Conventions Reminder

- Go 1.26, tabs for indentation, gofumpt formatted
- stdlib testing (no testify), slog for logging
- Event struct field is `TS` (not `Ts`) — `json:"ts"`
- Cobra commands in top-level `cmd/` package
- Business logic in `internal/` packages
- Every ticket mutation = markdown write + JSONL event
- `AppendSection(t, heading, actor, session, content, fields, ts)` for ticket body sections
