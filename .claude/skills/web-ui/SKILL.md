---
name: web-ui
description: Review or build web UI features using the templ + HTMX + DaisyUI stack. Covers component patterns, HTMX swap mechanics, SSE integration, handler structure, and DaisyUI conventions. Use when working on .templ files, web handlers, or SSE code.
argument-hint: "[file-or-feature-description]"
allowed-tools: Read, Glob, Grep, Bash, Edit, Write
---

# Web UI — templ + HTMX + DaisyUI

Work on the web UI at `$ARGUMENTS`. Before making changes, read existing code in `internal/web/` to follow established patterns.

## templ Component Patterns

### Three-Tier Template Structure

Every page follows a three-tier pattern:

1. **Page** — full HTML with layout wrapper, served on direct navigation
2. **Partial** — self-refreshing wrapper with SSE trigger, served on `/partials/` routes
3. **Content** — pure content component, served for filter/sort swaps

```templ
// Tier 1: Full page
templ BoardPage(data BoardData) {
	@Layout("Board", "/board", data.Project, data.Projects) {
		@BoardPartial(data)
	}
}

// Tier 2: SSE self-refresh wrapper
templ BoardPartial(data BoardData) {
	<div
		hx-get="/partials/board"
		hx-trigger="sse:refresh-work"
		hx-target="this"
		hx-swap="outerHTML"
	>
		@BoardContent(data)
	</div>
}

// Tier 3: Pure content (no SSE wrapper)
templ BoardContent(data BoardData) {
	// Actual UI content here
}
```

**When to use each tier:**
- Adding a new page? Create all three tiers.
- Adding a filter/sort control? Target the Content tier only.
- Adding real-time updates? Wire SSE trigger on the Partial tier.

### Props & Data Flow

- Pass data via structs, not individual parameters: `BoardData`, `ListData`, `TicketFormData`
- Define helper functions as Go functions at the top of the file, not inline in templates
- Use templ's `if` syntax for conditionals, not Go template `{{if}}`

### OOB (Out-of-Band) Swaps

Use `hx-swap-oob="true"` to update separate page regions from a single response:

```templ
templ TicketFormModalPartial(data TicketFormData) {
	<form hx-post={ formAction(data) } hx-target="#ticket-modal-body" hx-swap="innerHTML">
		// form fields
	</form>
	<div id="ticket-modal-header" hx-swap-oob="true">
		<h2>{ formTitle(data.Mode) }</h2>
	</div>
}
```

## HTMX Patterns

### Request Attributes

```html
hx-get="/partials/board"           <!-- fetch partial -->
hx-target="#content"               <!-- swap target (main container) -->
hx-target="this"                   <!-- swap self (for SSE refresh) -->
hx-swap="outerHTML"                <!-- replace wrapper div -->
hx-swap="innerHTML"                <!-- replace inner content only -->
hx-push-url="/list"                <!-- update browser URL -->
hx-push-url="false"               <!-- don't update URL (modals) -->
hx-disinherit="hx-swap"           <!-- prevent inherited swap mode -->
hx-include="[name='project']"     <!-- include hidden fields in request -->
```

### SSE Triggers

All real-time updates use a single unified `/events` SSE endpoint to avoid HTTP/1.1 connection limits (browsers allow ~6 per origin).

```html
<!-- On <body>: -->
hx-ext="sse"
sse-connect="/events"

<!-- On components: -->
hx-trigger="sse:refresh-work"      <!-- triggers on work-related events -->
hx-trigger="sse:refresh-activity"  <!-- triggers on activity events -->
```

### Modal Pattern

Single modal element in layout, swapped by HTMX:

```html
<!-- Trigger: -->
hx-get="/partials/form/new"
hx-target="#ticket-modal-body"
hx-swap="innerHTML"
hx-push-url="false"

<!-- Response closes modal via header (no body needed): -->
w.Header().Set("HX-Trigger", "closeModal")
```

### No fetch() Calls

All interactions use HTMX declarative attributes. Do not use `fetch()`, `XMLHttpRequest`, or direct DOM manipulation for data loading. The only JavaScript should be for:
- Modal open/close management
- Audio alerts
- Session activity dot animations
- Guard-once patterns to prevent duplicate listeners

## Go Handler Patterns

### Page vs Partial

```go
// Full page — direct navigation
func (h *Handler) Board(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildBoardData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.BoardPage(data).Render(r.Context(), w)
}

// Partial — HTMX swap (SSE refresh or filter change)
func (h *Handler) PartialBoard(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildBoardData(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = templates.BoardPartial(data).Render(r.Context(), w)
}
```

### HTMX-Aware Form Handling

```go
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	isHTMX := r.Header.Get("HX-Request") == "true"

	if err := r.ParseForm(); err != nil {
		if isHTMX {
			h.renderFormModalError(w, r, "new", "", err.Error())
		} else {
			h.renderFormError(w, r, "new", "", err.Error())
		}
		return
	}

	// ... create ticket ...

	if isHTMX {
		w.Header().Set("HX-Trigger", "closeModal")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/ticket/"+tk.ID, http.StatusSeeOther)
}
```

### Data Building

Extract query params and build data structs in separate helper methods:

```go
func (h *Handler) buildListData(r *http.Request) (templates.ListData, error) {
	q := r.URL.Query()
	project := q.Get("project")
	status := q.Get("status")
	sort := q.Get("sort")
	// ... filter, sort, build struct ...
}
```

## SSE Patterns

### Event Types

| Event | Purpose | HTMX Trigger |
|-------|---------|--------------|
| `refresh-work` | Ticket state changed | `sse:refresh-work` |
| `refresh-activity` | New activity logged | `sse:refresh-activity` |
| `ping` | Agent heartbeat | Custom JS handler |

### Server-Side Format

```go
fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, jsonData)
rc.Flush()
```

### Keepalive

Send comment-only keepalive every 5 seconds:
```go
fmt.Fprintf(w, ": keepalive\n\n")
```

## DaisyUI + Tailwind Conventions

### Theme

- Theme set on `<html>` via `data-theme="black"`
- Custom CSS variables: `--background`, `--foreground`, `--primary`, `--st-border`
- Use `hsl(var(--background))` syntax for custom variable colors

### Common Component Classes

```
btn btn-primary btn-sm          <!-- buttons -->
select select-sm                <!-- dropdowns -->
badge badge-sm                  <!-- status/priority badges -->
table table-sm                  <!-- data tables -->
card card-xs bg-base-200        <!-- cards -->
modal                           <!-- modals (use <dialog>) -->
alert alert-error               <!-- error messages -->
```

### Spacing & Layout

```
px-4 sm:px-6                    <!-- responsive horizontal padding -->
flex flex-wrap items-center gap-3  <!-- filter bars -->
grid grid-cols-1 sm:grid-cols-4    <!-- responsive form grids -->
```

### Status Colors (Custom CSS)

Status and priority use custom CSS classes, not inline Tailwind:
```
.st-status-open        { border-color: #3b82f6; }
.st-status-in-progress { border-color: #f59e0b; }
.st-status-done        { border-color: #22c55e; }
.st-priority-p0        { background: #dc2626; }
```

Define these as Go helper functions that return class strings:
```go
func statusBadgeClass(s string) string { ... }
func priorityClass(p string) string { ... }
```

## Rules

- **Follow the three-tier pattern** for any new page or major component.
- **Never use fetch() or direct DOM manipulation** for data loading — use HTMX attributes.
- **Check `HX-Request` header** in handlers that serve both full pages and HTMX partials.
- **Use a single SSE endpoint** — do not create additional SSE connections.
- **Use DaisyUI components** before reaching for custom CSS.
- **Custom CSS classes** should be prefixed with `st-` to avoid conflicts.
- **Guard once-patterns** in JavaScript to prevent duplicate event listeners on re-renders.
- **Responsive first**: use `sm:` breakpoint for desktop, mobile is the default.
