package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
	"github.com/boozedog/smoovtask/internal/web/templates"
)

// Projects renders the projects page.
func (h *Handler) Projects(w http.ResponseWriter, r *http.Request) {
	data := h.buildProjectsData(r)
	_ = templates.ProjectsPage(data).Render(r.Context(), w)
}

// PartialProjects renders the projects partial for htmx swaps.
func (h *Handler) PartialProjects(w http.ResponseWriter, r *http.Request) {
	data := h.buildProjectsData(r)
	_ = templates.ProjectsPartial(data).Render(r.Context(), w)
}

func (h *Handler) buildProjectsData(r *http.Request) templates.ProjectsData {
	// Merge registered projects from config with ticket-discovered ones.
	projectNames := make(map[string]bool)
	for name := range h.cfg.Projects {
		projectNames[name] = true
	}
	for _, name := range h.allProjects() {
		projectNames[name] = true
	}

	// Get all tickets once.
	allTickets, _ := h.store.ListMeta(ticket.ListFilter{})

	// Group tickets by project.
	ticketsByProject := make(map[string][]*ticket.Ticket)
	for _, tk := range allTickets {
		ticketsByProject[tk.Project] = append(ticketsByProject[tk.Project], tk)
	}

	// Get recent events for session counting + last activity.
	const recentLimit = 500
	events := recentEvents(h.eventsDir, event.Query{}, recentLimit)

	// Build per-project event data.
	type projectEventInfo struct {
		activeSessions map[string]bool // runIDs with recent hook activity
		lastActivity   time.Time
	}
	projectEvents := make(map[string]*projectEventInfo)
	now := time.Now().UTC()
	const activeThreshold = 10 * time.Minute

	for _, ev := range events {
		proj := ev.Project
		if proj == "" {
			continue
		}
		info, ok := projectEvents[proj]
		if !ok {
			info = &projectEventInfo{activeSessions: make(map[string]bool)}
			projectEvents[proj] = info
		}
		if ev.TS.After(info.lastActivity) {
			info.lastActivity = ev.TS
		}
		if ev.RunID != "" && strings.HasPrefix(ev.Event, "hook.") {
			if now.Sub(ev.TS) <= activeThreshold {
				info.activeSessions[ev.RunID] = true
			}
		}
	}

	var summaries []templates.ProjectSummary
	for name := range projectNames {
		s := templates.ProjectSummary{
			Name:              name,
			TicketsByStatus:   make(map[ticket.Status]int),
			TicketsByPriority: make(map[ticket.Priority]int),
		}

		// Config info.
		if pc, ok := h.cfg.Projects[name]; ok {
			s.Path = pc.Path
			s.Repo = pc.Repo
		}

		// Ticket stats.
		for _, tk := range ticketsByProject[name] {
			s.TotalTickets++
			s.TicketsByStatus[tk.Status]++
			s.TicketsByPriority[tk.Priority]++

			if tk.Status == ticket.StatusDone {
				s.DoneCount++
			}

			// Oldest open ticket age.
			if tk.Status != ticket.StatusDone && tk.Status != ticket.StatusCancelled {
				age := now.Sub(tk.Created)
				if s.OldestOpenAge == 0 || age > s.OldestOpenAge {
					s.OldestOpenAge = age
				}
			}
		}

		// Event info.
		if info, ok := projectEvents[name]; ok {
			s.ActiveSessions = len(info.activeSessions)
			s.LastActivity = info.lastActivity
		}

		// Worktrees.
		if s.Path != "" {
			expanded, err := expandHome(s.Path)
			if err == nil {
				s.Worktrees = scanWorktrees(expanded)
			}
		}

		summaries = append(summaries, s)
	}

	// Sort: projects with more active sessions first, then alphabetically.
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].ActiveSessions != summaries[j].ActiveSessions {
			return summaries[i].ActiveSessions > summaries[j].ActiveSessions
		}
		return summaries[i].Name < summaries[j].Name
	})

	allProj := make([]string, 0, len(projectNames))
	for name := range projectNames {
		allProj = append(allProj, name)
	}
	sort.Strings(allProj)

	return templates.ProjectsData{
		Projects:       summaries,
		CurrentProject: r.URL.Query().Get("project"),
		AllProjects:    allProj,
	}
}

func scanWorktrees(projectPath string) []templates.WorktreeInfo {
	wtDir := filepath.Join(projectPath, ".worktrees")
	entries, err := os.ReadDir(wtDir)
	if err != nil {
		return nil
	}

	var result []templates.WorktreeInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		result = append(result, templates.WorktreeInfo{
			Name:     name,
			Path:     filepath.Join(wtDir, name),
			IsTicket: strings.HasPrefix(name, "st_"),
		})
	}
	return result
}

func expandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}
