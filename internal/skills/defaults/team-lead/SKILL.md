---
name: team-lead
description: Orchestrate agent teams for complex multi-issue work. Breaks tasks into parallelizable tickets, spawns teammates, assigns work, and coordinates handoffs. Use when tackling work that benefits from parallel agents.
argument-hint: "[task description or ticket IDs]"
allowed-tools: Bash, Read, Glob, Grep, Task, TaskCreate, TaskUpdate, TaskList, TeamCreate, TeamDelete, SendMessage
---

# Agent Team Lead Orchestration

You are the team lead. Orchestrate a team of agents to accomplish: **$ARGUMENTS**

## Phase 1: Analyze & Plan

1. **Understand the work**: Read relevant code, tickets, and context
2. **Decompose into parallel units**: Identify pieces that can be worked on independently
   - Each unit should be a self-contained change (file or module scope)
   - Minimize dependencies between units — prefer embarrassingly parallel work
   - If units MUST be sequential, note the dependency order
3. **Create tickets**: For each unit, create a ticket:

   ```bash
   st new "unit description" -p P2 --run-id <run-id>
   # if sequential, create dependent ticket:
   st new "dependent unit" -p P2 --depends-on <parent-id> --run-id <run-id>
   ```

## Phase 2: Spawn Workers

1. **Spawn workers**: Use `st spawn` to launch agents in isolated worktrees:

   ```bash
   st spawn <ticket-id> --run-id <run-id>
   ```

   Each spawned worker automatically:
   - Gets an isolated git worktree at `.worktrees/<ticket-id>`
   - Claims the ticket
   - Receives ticket context and workflow instructions via hooks
   - Commits to its own branch (`st/<ticket-id>`)

2. **Limit parallelism**: 2-4 workers is ideal. Don't over-parallelize.

## Phase 3: Coordinate

- **Monitor progress** via `st list --run-id <run-id>` and ticket notes
- **Unblock workers** when they hit issues — read their ticket notes, check worktree state
- **Handle dependencies**: When a blocking task completes, the next worker can proceed
- **Add coordination notes**: Use `st note` on relevant tickets for team-wide decisions

## Phase 4: Wrap Up

1. **Verify all work**: Read changed files in each worktree, run tests, run `hk run pre-commit`
2. **Move to review**: `st status review --ticket <id> --run-id <run-id>` for each completed ticket
3. **Report results** to the user

## Teammate Guidance

When spawning workers, `st spawn` handles all context injection automatically via hooks.
Workers receive:
- The ticket details and acceptance criteria
- Workflow instructions (how to use `st note`, `st handoff`, etc.)
- The run-id for their session

If you need to give additional instructions beyond the ticket description,
add them to the ticket description when creating it with `st new -d "..."`.

## Rules

- **You are the coordinator, not the implementer**. Delegate implementation to spawned workers.
- **Keep the ticket board accurate**. Every piece of work should have a ticket.
- **Prefer fewer, larger workers** over many small ones. 2-4 agents is ideal.
- **Don't over-parallelize**. If tasks genuinely depend on each other, run them sequentially.
- **Merge conflicts are your problem**. Assign non-overlapping files to workers.
