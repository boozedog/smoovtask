# smoovtask

AI agent workflow and ticketing system for Claude Code. Command: `st`

## Tech Stack

Go 1.26 · Cobra (CLI) · BurntSushi/toml (config) · YAML v3 (ticket frontmatter) · JSONL (event log)

**Web UI:** templ (Go HTML templating) · HTMX with SSE · FrankenUI. Avoid direct DOM manipulation — prefer properly structured templ partials that HTMX can swap/process correctly.

## Architecture

`st CLI → internal packages → markdown tickets (Obsidian vault) + JSONL events (~/.smoovtask/)`

Tickets are markdown files with YAML frontmatter in an Obsidian vault. Events are append-only JSONL logs rotated daily. Claude Code hooks call `st hook <event>` for context injection and activity logging.

## Design Principle

`st` is fully self-contained. All workflow instructions are injected at runtime via hooks — never rely on CLAUDE.md, memory files, or skills to convey workflow guidance to the agent. If the agent needs to know something, `st` tells it directly.

## Conventions

- **Tabs** for indentation (gofumpt enforced)
- **stdlib testing** (no testify)
- **slog** for logging
- Cobra commands in top-level `cmd/` package
- Business logic in `internal/` packages
- Formatter is **gofumpt** (not gofmt) — use `just fmt`
- After modifying `st` source code, run `just install` to update the binary in your PATH. The `st` hooks run the installed binary, not the local build — stale binaries cause confusing behavior.

## Reference

- [Project structure](docs/structure.md)
- [Building & running](docs/building.md)
- [Storage paths](docs/storage.md)
- Franken UI contexts snapshot (local): `docs/franken-ui/contexts/`
- Snapshot notes and refresh script: `docs/franken-ui/README.md`, `docs/franken-ui/update-contexts.sh`
