---
name: review-team-lead
description: Find all reviewable tickets and orchestrate a team of agents to review them. Agents read code, check diffs, run tests, and approve or reject each item. Use when there are multiple items awaiting review.
argument-hint: "[optional: specific ticket IDs to review]"
allowed-tools: Bash, Read, Glob, Grep, Task, TaskCreate, TaskUpdate, TaskList, TeamCreate, TeamDelete, SendMessage
---

# Review Team Lead

Orchestrate a team of agents to review tickets that are awaiting review.

Scope: **$ARGUMENTS** (default: all reviewable tickets)

## Phase 1: Discover Reviewable Work

1. **Find reviewable tickets**:

   ```bash
   st list --status review --run-id <run-id>
   ```

   If `$ARGUMENTS` specifies ticket IDs, filter to just those.

2. **Gather context for each ticket**: For every reviewable ticket, run:

   ```bash
   st show <id> --run-id <run-id>
   ```

   Understand what was done, what changed, and what the acceptance criteria are.

3. **Triage**: Group tickets by complexity:
   - **Trivial** — typos, config tweaks, minor formatting (can batch-review yourself)
   - **Standard** — single-scope changes with clear diffs (one agent per ticket)
   - **Complex** — multi-file changes, architectural decisions (one agent, more context)

   If there are only 1-2 trivial items, skip the team and review them directly.

## Phase 2: Set Up the Review Team

1. **Create the team**: Use TeamCreate with a descriptive name (e.g., `review-batch-<date>`)

2. **Spawn reviewer agents**: Use the Task tool with `team_name` parameter
   - Use `subagent_type: "general-purpose"` for all reviewers
   - Each agent gets a clear review assignment (see prompt template below)
   - **2-4 agents max** — batch multiple trivial tickets per agent if needed
   - Assign non-overlapping tickets to each agent

## Phase 3: Coordinate Reviews

- **Monitor progress** via TaskList and messages from teammates
- **Answer questions** — if an agent is uncertain about intent, check the ticket notes or ask the user
- **Aggregate findings** — collect approve/reject recommendations from agents
- **Resolve disagreements** — if an agent flags something ambiguous, investigate yourself

## Phase 4: Act on Results

For each reviewed ticket, claim it for review first:

```bash
st review <id> --run-id <run-id>
```

Then act on the reviewer's findings:

- **Approve** (if absolutely certain of correctness):

  ```bash
  st status done --run-id <run-id>
  ```

- **Send to human review** (default — when in any doubt):

  ```bash
  st status human-review --run-id <run-id>
  ```

- **Reject** (if problems need fixing):

  ```bash
  st status rework --run-id <run-id>
  ```

  Add a note with specific findings first:

  ```bash
  # Write findings to a file, then:
  st note --file findings.md --ticket <id> --run-id <run-id>
  ```

## Phase 5: Wrap Up

1. **Summarize results** to the user: how many approved, sent to human review, rejected, and why
2. **Shut down teammates**: Send shutdown_request to each teammate
3. **Clean up team**: TeamDelete after all teammates have shut down

## Reviewer Agent Prompt Template

When spawning a reviewer agent, include this context:

```text
You are reviewing ticket <ID>: "<title>"

Context:
  <paste st show output and relevant notes>

Your review process:
  1. Read the ticket details and notes to understand what was done
  2. Read ALL changed/relevant files mentioned in the ticket
  3. Check the diff in the worktree: cd .worktrees/<ID> && git diff main...HEAD
  4. Run tests if applicable: `go test ./path/to/package/...` (or appropriate test command)
  5. Run linters if applicable: `hk run pre-commit`
  6. Evaluate against this checklist:
     - Does the change do what the ticket describes?
     - Is the code correct and free of obvious bugs?
     - Are there security concerns?
     - Are edge cases handled?
     - Are tests adequate for the change?
     - Does it follow project conventions?

When done, send me a message with your verdict:
  - APPROVE: No issues found (or only minor style nits that don't warrant rejection)
  - HUMAN-REVIEW: Looks good but needs human sign-off (complex changes, architectural decisions)
  - REJECT: Specific problems that must be fixed, with file:line references

Important:
  - Do NOT edit any files — this is a read-only review
  - Do NOT run st status commands yourself — report findings to me
  - If you can't determine the scope of changes, message me with "unclear scope"
```

## Rules

- **You are the coordinator**. Agents review, you make the final approve/reject call.
- **Never approve your own work**. If a ticket was implemented in this session, skip it.
- **Read before delegating**. Skim each ticket yourself to write good reviewer prompts — don't blindly delegate.
- **Default to human-review**. Use `st status done` only when you can fully verify correctness yourself. Use `st status human-review` when in any doubt.
- **Be conservative**. When in doubt, reject with specific feedback rather than approving questionable work.
