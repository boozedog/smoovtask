# Project Structure

- `cmd/st/` — Entry point (`main.go`)
- `cmd/` — CLI commands (Cobra): root, init, new, list, show, pick, status, note, review, hook
- `internal/config/` — TOML config loading, project registry
- `internal/ticket/` — Ticket struct, ID generation, markdown parse/write, file-based store
- `internal/event/` — JSONL event log: append (flock), daily rotation, query/filter
- `internal/workflow/` — State machine, transition rules, validation
- `internal/project/` — Project detection from PWD
- `internal/identity/` — Invocation identity (`--run-id` for agents, `--human` for manual use)
- `internal/hook/` — Hook command handlers (session-start, pre/post-tool, subagent, stop)
