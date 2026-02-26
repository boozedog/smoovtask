# Web UI Fixes: Activity Double-Render, Dark Theme, Partial Endpoint

Date: 2026-02-26

## Problems

### 1. Activity page double-rendering on filter change

The activity filter `<select>` elements used `hx-get="/activity"` with `hx-target="body"` and `hx-swap="outerHTML"`. When a filter changed, htmx replaced the entire `<body>` with a full page response (nav + content), causing the layout to re-render inside itself.

Board and list pages already used partial endpoints correctly (e.g. `hx-get="/partials/list"` targeting a content div with `innerHTML` swap).

### 2. Dark theme not activating

The `<html>` element had `class="dark"` but Franken UI 2.0 requires a compound selector — dark mode styles are scoped under `.dark.uk-theme-zinc` (or other theme names). Without a theme class, the CSS variables for dark colors were never set.

Additionally, the `<body>` used classes `uk-background-default` and `uk-text-default` which don't exist in the bundled Franken UI build. The theme sets CSS custom properties (`--background`, `--foreground`, `--border`, `--card`, `--primary`) but nothing applied them to the body element.

Custom CSS also referenced non-existent variables like `--uk-border`, `--uk-card-default-background`, and `--uk-primary` with hardcoded fallbacks, bypassing the theme system entirely.

### 3. No `/partials/activity` route

Unlike board (`/partials/board`) and list (`/partials/list`), activity had no partial endpoint, so there was no way to fetch just the content fragment for htmx swaps.

### 4. SSE refresh overwrote filtered results (discovered during testing)

The SSE wrapper div used a static `hx-get` URL built at render time via `partialActivityURL()`. This URL only included filters from the initial page load. When SSE events arrived, the refresh fetched `/partials/activity?` with no filter params, wiping out any client-side filter selection.

## Changes

### `internal/web/handler/activity.go`

Added `PartialActivity` handler mirroring the `PartialList` pattern — calls `buildActivityData()` then renders only `ActivityContent` (no layout wrapper):

```go
func (h *Handler) PartialActivity(w http.ResponseWriter, r *http.Request) {
    data, err := h.buildActivityData(r)
    // ...
    templates.ActivityContent(data).Render(r.Context(), w)
}
```

### `internal/web/server.go`

Added route registration:

```go
mux.HandleFunc("GET /partials/activity", h.PartialActivity)
```

### `internal/web/templates/activity.templ`

**Filter selects** (both project and event_type):
- `hx-get="/activity"` -> `hx-get="/partials/activity"`
- `hx-target="body"` -> `hx-target="#activity-content"`
- `hx-swap="outerHTML"` -> `hx-swap="innerHTML"`
- Removed `hx-push-url="true"` (partials shouldn't push URL)

**SSE wrapper div:**
- Replaced static `hx-get={ partialActivityURL(data.Filter) }` with `hx-get="/partials/activity"`
- Added `hx-include="[name='project'],[name='event_type']"` so SSE-triggered refreshes include current filter values from the dropdowns

**Removed** the `partialActivityURL()` helper function (no longer needed).

### `internal/web/templates/layout.templ`

**Dark theme activation:**
- `<html>` class: `"dark"` -> `"dark uk-theme-zinc"`

**Body styling:**
- Replaced `class="uk-background-default uk-text-default"` with inline `style="background-color: hsl(var(--background)); color: hsl(var(--foreground));"` since the utility classes don't exist in this Franken UI build

**CSS variable fixes** (all occurrences):
- `var(--uk-border, #333)` -> `hsl(var(--border))`
- `var(--uk-card-default-background, #1a1a1a)` -> `hsl(var(--card))`
- `var(--uk-primary, #1e87f0)` -> `hsl(var(--primary))`

### `internal/web/handler/handler_test.go`

Added two tests:

- `TestPartialActivity` — verifies the partial endpoint returns 200, contains event data, and does NOT include `<!DOCTYPE html>` (confirming no layout wrapper)
- `TestActivityFilterByEventType` — creates a `hook.session_start` event alongside the existing `ticket.created` event, filters via `?event_type=ticket`, and asserts only ticket events appear

## Verification

- `go build ./...` — compiles cleanly
- `go test ./...` — all tests pass (15 handler tests including 2 new)
- Browser testing confirmed:
  - All pages (board, list, activity) render with dark background
  - Activity filter dropdowns swap only `#activity-content` (no double-render)
  - SSE live refresh respects active filter selections
  - Ticket cards, priority badges, borders all use theme colors correctly
