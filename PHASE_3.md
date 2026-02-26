# smoovtask Phase 3: Core Completion + Test Coverage

## Context for the Next Agent

Phase 2 is complete and pushed. You now have:

- **Hook Install**: `st hooks install` merges all 10 hooks into `~/.claude/settings.json`
- **Workflow Commands**: `hold`, `unhold`, `assign`, `close` + `--depends-on` on `new`
- **Dependency System**: `CheckDependencies`, `FindDependents`, `AutoUnblock` in `internal/ticket/deps.go`
- **Priority Sorting**: Score-based batch selection in session-start (priority weight + REVIEW boost)
- **All Hook Handlers**: session-start, subagent-start, pre/post-tool, stop, subagent-stop, task-completed, teammate-idle, permission-request, session-end
- **Refactored resolver**: `resolveCurrentTicket` extracted to `cmd/helpers.go`

All tests pass, lint is clean.

Read `DESIGN.md` for the full spec. Read `PHASE_2.md` for what was built in Phase 2.

## What Phase 3 Covers

Two goals: finish the remaining core commands, and add test coverage for the entire CLI layer.

### Step 1: `st override` Command

**Files:**
- `cmd/override.go` — `st override <ticket-id> <status>`

**Behavior (from DESIGN.md):**
- Human-only force override — bypasses all transition rules
- Sets ticket to any valid status, no validation
- Clears `PriorStatus` (override is a clean slate)
- AppendSection with heading "Override", actor "human", content showing from → to
- Log event: `"status.override"` with data `{"from": oldStatus, "to": newStatus, "reason": "human-override"}`
- Print: "Override st_xxxxxx: OLD → NEW"

Use `workflow.StatusFromAlias` to resolve the target status (so `st override st_xxx done` works).

### Step 2: `st context` Command

**Files:**
- `cmd/context.go` — `st context`

**Behavior (from DESIGN.md):**
- Print current session context for debugging/hooks
- Output JSON with: `session_id`, `project` (from PWD), `active_ticket` (if any), `cwd`
- If no active ticket, `active_ticket` is null
- Useful for agents and humans to understand what st thinks the current state is

### Step 3: CLI Test Infrastructure

**Files:**
- `cmd/cmd_test.go` — shared test helpers for all cmd tests

**Test helper pattern:**
The internal packages use `t.TempDir()` with real file I/O. Follow the same pattern for cmd tests. Create a helper that sets up a complete smoovtask environment:

```go
// testEnv sets up a temp config, tickets dir, and events dir.
// It sets env vars so config.Load() finds the test config.
// Returns cleanup function.
type testEnv struct {
    ConfigDir  string
    TicketsDir string
    EventsDir  string
    Store      *ticket.Store
    EventLog   *event.EventLog
    Config     *config.Config
}

func newTestEnv(t *testing.T) *testEnv
```

Key requirements:
- Create temp `~/.smoovtask/` equivalent with `config.toml`
- Register a test project pointing to a temp directory
- Set `SMOOVBRAIN_CONFIG` env var (or equivalent) so `config.Load()` uses test config
- Provide helpers to create test tickets in known states
- Set/unset `CLAUDE_SESSION_ID` for session-based tests

**Important**: Check how `config.Load()` resolves its path. If it hardcodes `~/.smoovtask/config.toml`, you may need to add a `config.LoadFrom(path)` variant or use an env var override. The config package already has `LoadFrom` — verify and use it. You'll need to wire the cmd layer to support this (e.g., via a package-level config path override or root command persistent flag).

### Step 4: Command Tests

**Files:**
- `cmd/new_test.go` — test ticket creation, --depends-on auto-block, --priority, --tags
- `cmd/pick_test.go` — test pick by ID, pick auto-select, transition validation
- `cmd/status_test.go` — test transitions, auto-unblock on DONE, resolver logic
- `cmd/note_test.go` — test note append, resolver with --ticket flag
- `cmd/review_test.go` — test review eligibility, session disqualification
- `cmd/hold_test.go` — test hold sets BLOCKED + prior-status, already-blocked error
- `cmd/unhold_test.go` — test unhold snaps back, not-blocked error, nil prior-status error
- `cmd/assign_test.go` — test assignee update
- `cmd/close_test.go` — test force-DONE, auto-unblock dependents
- `cmd/override_test.go` — test force any transition, alias resolution
- `cmd/list_test.go` — test filters (--project, --status)
- `cmd/show_test.go` — test ticket display
- `cmd/hooks_install_test.go` — test merge logic, idempotency, preserves existing settings

**What to test per command:**
1. Happy path — command succeeds, ticket state is correct after
2. Validation errors — wrong status, missing args, ticket not found
3. Event logging — JSONL event written with correct fields
4. Markdown body — AppendSection added with right heading/actor/content
5. Edge cases specific to each command (see notes below)

**Per-command edge cases:**

| Command | Edge Cases |
|---------|-----------|
| `new` | --depends-on with all-DONE deps (no auto-block), --depends-on with unresolved (auto-block), missing dep ID |
| `pick` | Already IN-PROGRESS, no open tickets, pick from REWORK |
| `status` | All valid transitions, invalid transitions, auto-unblock chain |
| `review` | Session touched ticket (denied), clean session (allowed), ticket not in REVIEW |
| `hold` | Already BLOCKED |
| `unhold` | Not BLOCKED, nil PriorStatus |
| `close` | From any status, auto-unblocks dependents |
| `override` | Any status → any status, alias resolution |
| `hooks install` | Fresh install, idempotent re-install, existing non-st hooks preserved |

### Step 5: Hook Handler Tests

**Files:**
- `internal/hook/subagent_stop_test.go`
- `internal/hook/task_completed_test.go`
- `internal/hook/teammate_idle_test.go`
- `internal/hook/permission_request_test.go`
- `internal/hook/session_end_test.go`
- `internal/hook/pre_tool_test.go`
- `internal/hook/post_tool_test.go`
- `internal/hook/stop_test.go`

These are all simple log-only handlers. Tests should verify:
1. Event is written to JSONL with correct event type
2. Project detection from CWD works
3. Session ID is propagated
4. Config errors don't cause failures (return nil)
5. Permission request returns empty Output (no decision)

Follow the pattern from `session_start_test.go` and `subagent_start_test.go`.

## Parallelization

- **Steps 1 + 2** are independent (new command files, no overlap)
- **Step 3** should be done first or early — other test steps depend on the test helpers
- **Steps 4 + 5** can run in parallel once Step 3 is done, but Step 4 files overlap with Steps 1+2 (override_test.go needs override.go)

Recommended order:
1. Steps 1 + 2 in parallel (small, quick)
2. Step 3 (test infrastructure)
3. Steps 4 + 5 in parallel (bulk of work)

Or as 3 agents:
- **Agent A**: Steps 1 + 2 (core commands) — then Step 3 (test infra)
- **Agent B**: Step 4 command tests (after Agent A finishes Step 3)
- **Agent C**: Step 5 hook handler tests (independent, can start immediately)

## Verification

After all steps:
1. `go test ./...` — all pass, including new cmd/ tests
2. `go test -cover ./...` — check coverage percentages
3. `golangci-lint run ./...` — clean
4. `st override st_xxx done` works from any status
5. `st context` outputs valid JSON with session info
6. No regressions in existing functionality

## Conventions Reminder

- Go 1.26, tabs for indentation, gofumpt formatted
- stdlib testing (no testify), slog for logging
- Event struct field is `TS` (not `Ts`) — `json:"ts"`
- Cobra commands in top-level `cmd/` package
- Business logic in `internal/` packages
- Every ticket mutation = markdown write + JSONL event
- `AppendSection(t, heading, actor, session, content, fields, ts)` for ticket body sections
- Test pattern: `t.TempDir()` for isolation, real file I/O, no mocks
