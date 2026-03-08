# Project Structure

- `cmd/st/` — Entry point (`main.go`)
- `cmd/` — CLI commands (Cobra): root, init, new, list, show, pick, status, note, review, leader, work, launch, spawn, hook, install, uninstall, assign, hold, unhold, close, cancel, handoff, override, context, web, prep
- `internal/config/` — TOML config loading, project registry
- `internal/ticket/` — Ticket struct, ID generation, markdown parse/write, file-based store, dependency graph
- `internal/event/` — JSONL event log: append (flock), daily rotation, query/filter
- `internal/workflow/` — State machine, transition rules, review eligibility, note requirements
- `internal/project/` — Project detection from PWD, git remote matching
- `internal/identity/` — Invocation identity (`--run-id` for agents, `--human` for manual use)
- `internal/hook/` — Hook command handlers (10 event types: session-start, pre/post-tool, subagent start/stop, permission-request, task-completed, teammate-idle, stop, session-end)
- `internal/spawn/` — Multi-agent orchestration: backend interface (Claude/OpenCode/PI), worktree management, prompt building, worker status, tmux integration
- `internal/guidance/` — Centralized workflow instructions for context injection (implementation vs review roles)
- `internal/rules/` — Tool-use policy evaluation: bash allowlists, git safety, file protection, pipeline restrictions. Includes embedded default YAML rule files
- `internal/web/` — Web UI server
  - `handler/` — HTTP route handlers (board, list, ticket detail, activity feed, agents, critical path)
  - `middleware/` — CORS, rate limiting
  - `sse/` — Server-Sent Events broker and fsnotify file watcher
  - `static/` — Embedded assets (DaisyUI CSS, Tailwind CSS, htmx, fonts) via go:embed
  - `templates/` — templ Go HTML templates (layout, board, list, ticket, activity, components, forms)
