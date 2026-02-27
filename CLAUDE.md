# smoovtask

AI agent workflow and ticketing system for Claude Code. Command: `st`

## Tech Stack

Go 1.26 · Cobra (CLI) · BurntSushi/toml (config) · YAML v3 (ticket frontmatter) · JSONL (event log)

## Architecture

```
st CLI → internal packages → markdown tickets (Obsidian vault) + JSONL events (~/.smoovtask/)
```

Tickets are markdown files with YAML frontmatter in an Obsidian vault. Events are append-only JSONL logs rotated daily. Claude Code hooks call `st hook <event>` for context injection and activity logging.

## Project Structure

- `cmd/st/` — Entry point (`main.go`)
- `cmd/` — CLI commands (Cobra): root, init, new, list, show, pick, status, note, review, hook
- `internal/config/` — TOML config loading, project registry
- `internal/ticket/` — Ticket struct, ID generation, markdown parse/write, file-based store
- `internal/event/` — JSONL event log: append (flock), daily rotation, query/filter
- `internal/workflow/` — State machine, transition rules, validation
- `internal/project/` — Project detection from PWD
- `internal/identity/` — Session ID from `CLAUDE_SESSION_ID` env
- `internal/hook/` — Hook command handlers (session-start, pre/post-tool, subagent, stop)

## Building & Running

```sh
just install        # quick install from local source (use during dev)
just build          # templ generate + build + install
just test           # go test -v ./...
just test-short     # go test ./... (non-verbose)
just test-cover     # tests with HTML coverage report
just lint           # golangci-lint
just fmt            # gofumpt (formatter)
just vuln           # govulncheck
just release        # goreleaser snapshot build
just clean          # remove build artifacts
```

**IMPORTANT:** After modifying `st` source code, run `just install` to update the binary in your PATH. The `st` hooks run the installed binary, not the local build — stale binaries cause confusing behavior.

## Storage Paths

- Config: `~/.smoovtask/config.toml`
- Events: `~/.smoovtask/events/YYYY-MM-DD.jsonl`
- Tickets: `<vault_path>/tickets/YYYY-MM-DDTHH:MM-st_xxxxxx.md` (default vault: `~/obsidian/smoovtask`)

## Design Principle

`st` is fully self-contained. All workflow instructions are injected at runtime via hooks — never rely on CLAUDE.md, memory files, or skills to convey workflow guidance to the agent. If the agent needs to know something, `st` tells it directly.

## Conventions

- **Tabs** for indentation (gofumpt enforced)
- **stdlib testing** (no testify)
- **slog** for logging
- Cobra commands in top-level `cmd/` package
- Business logic in `internal/` packages
- Formatter is **gofumpt** (not gofmt) — use `just fmt`
