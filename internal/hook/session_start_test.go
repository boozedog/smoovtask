package hook

import (
	"strings"
	"testing"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/boozedog/smoovtask/internal/ticket"
)

func TestHandleSessionStartLogsEvent(t *testing.T) {
	projectPath := t.TempDir()
	env := setupTestEnv(t, projectPath)

	// Create a ticket so the handler has something to find
	store := ticket.NewStore(env.ticketsDir(t))
	tk := &ticket.Ticket{
		ID:       "st_test01",
		Title:    "Test ticket",
		Project:  "test-project",
		Status:   ticket.StatusOpen,
		Priority: ticket.PriorityP2,
		Created:  time.Now().UTC(),
		Updated:  time.Now().UTC(),
	}
	if err := store.Create(tk); err != nil {
		t.Fatalf("create ticket: %v", err)
	}

	input := &Input{
		SessionID: "sess-start-test",
		CWD:       projectPath,
	}

	_, err := HandleSessionStart(input)
	if err != nil {
		t.Fatalf("HandleSessionStart() error: %v", err)
	}

	ev := readTodayEvent(t, env.EventsDir)
	assertEvent(t, ev, event.HookSessionStart, "sess-start-test", "test-project")

	openCount, _ := ev.Data["open_count"].(float64)
	if openCount != 1 {
		t.Errorf("open_count = %v, want 1", ev.Data["open_count"])
	}
}

func TestHandleSessionStartNoProject(t *testing.T) {
	setupTestEnv(t, "")

	input := &Input{
		SessionID: "sess-no-proj",
		CWD:       "/some/unknown/path",
	}

	_, err := HandleSessionStart(input)
	if err != nil {
		t.Fatalf("HandleSessionStart() error: %v", err)
	}
}

func TestBuildBoardSummaryOpen(t *testing.T) {
	open := []*ticket.Ticket{
		{ID: "st_a7Kx2m", Title: "Add rate limiting", Priority: ticket.PriorityP2, Status: ticket.StatusOpen},
		{ID: "st_c1Dw4n", Title: "Fix CORS headers", Priority: ticket.PriorityP3, Status: ticket.StatusOpen},
	}

	summary := buildBoardSummary("api-server", "sess-abc123", open, nil)

	if !strings.Contains(summary, "smoovtask — api-server — 2 tickets ready") {
		t.Errorf("missing header, got:\n%s", summary)
	}
	if !strings.Contains(summary, "Run: sess-abc123") {
		t.Errorf("missing session ID, got:\n%s", summary)
	}
	if !strings.Contains(summary, "st_a7Kx2m") {
		t.Error("missing ticket ID st_a7Kx2m")
	}
	if !strings.Contains(summary, "st pick") {
		t.Error("missing pick instruction")
	}
	if !strings.Contains(summary, "st note") {
		t.Error("missing note instruction")
	}
	if !strings.Contains(summary, "st status review") {
		t.Error("missing status review instruction")
	}
	if !strings.Contains(summary, "Follow the user's instructions") {
		t.Error("missing user-primacy note")
	}
}

func TestBuildBoardSummaryReview(t *testing.T) {
	review := []*ticket.Ticket{
		{ID: "st_test01", Title: "Reviewed ticket", Priority: ticket.PriorityP2, Status: ticket.StatusReview},
	}

	summary := buildBoardSummary("proj", "", nil, review)

	if !strings.Contains(summary, "Review:") {
		t.Error("should show Review section when review tickets exist")
	}
	if !strings.Contains(summary, "st review") {
		t.Error("missing st review in workflow")
	}
	if !strings.Contains(summary, "st note") {
		t.Error("missing st note in workflow")
	}
	if !strings.Contains(summary, "st status done") {
		t.Error("missing st status done in workflow")
	}
	if !strings.Contains(summary, "st status rework") {
		t.Error("missing st status rework in workflow")
	}
	if !strings.Contains(summary, "Follow the user's instructions") {
		t.Error("missing user-primacy note")
	}
}

func TestBuildBoardSummaryEmpty(t *testing.T) {
	summary := buildBoardSummary("proj", "", nil, nil)
	if summary != "" {
		t.Errorf("expected empty summary, got: %q", summary)
	}
}

func TestBuildBoardSummaryReviewPreferred(t *testing.T) {
	open := []*ticket.Ticket{
		{ID: "st_open01", Title: "Open ticket", Priority: ticket.PriorityP3, Status: ticket.StatusOpen},
	}
	review := []*ticket.Ticket{
		{ID: "st_rev01", Title: "Review ticket", Priority: ticket.PriorityP2, Status: ticket.StatusReview},
	}

	summary := buildBoardSummary("proj", "", open, review)

	// Review P2 (45) beats Open P3 (30), so review section should appear first
	reviewIdx := strings.Index(summary, "Review:")
	openIdx := strings.Index(summary, "Open:")
	if reviewIdx == -1 {
		t.Fatal("missing Review section")
	}
	if openIdx == -1 {
		t.Fatal("missing Open section — both batches should be shown")
	}
	if reviewIdx > openIdx {
		t.Error("Review section should appear before Open section when review scores higher")
	}
}

func TestBuildBoardSummarySortedByPriority(t *testing.T) {
	open := []*ticket.Ticket{
		{ID: "st_low", Title: "Low priority", Priority: ticket.PriorityP4, Status: ticket.StatusOpen},
		{ID: "st_high", Title: "High priority", Priority: ticket.PriorityP1, Status: ticket.StatusOpen},
		{ID: "st_mid", Title: "Mid priority", Priority: ticket.PriorityP3, Status: ticket.StatusOpen},
	}

	summary := buildBoardSummary("proj", "", open, nil)

	highIdx := strings.Index(summary, "st_high")
	midIdx := strings.Index(summary, "st_mid")
	lowIdx := strings.Index(summary, "st_low")

	if highIdx == -1 || midIdx == -1 || lowIdx == -1 {
		t.Fatalf("missing ticket IDs in summary:\n%s", summary)
	}
	if highIdx > midIdx || midIdx > lowIdx {
		t.Errorf("tickets not sorted by priority (P1 < P3 < P4), got:\n%s", summary)
	}
}

func TestBuildBoardSummaryReviewBeatsSamePriority(t *testing.T) {
	// REVIEW gets +5 boost, so P3 REVIEW (35) beats P3 OPEN (30)
	open := []*ticket.Ticket{
		{ID: "st_open01", Title: "Open P3", Priority: ticket.PriorityP3, Status: ticket.StatusOpen},
	}
	review := []*ticket.Ticket{
		{ID: "st_rev01", Title: "Review P3", Priority: ticket.PriorityP3, Status: ticket.StatusReview},
	}

	summary := buildBoardSummary("proj", "", open, review)

	// Review should come first due to +5 boost
	reviewIdx := strings.Index(summary, "Review:")
	openIdx := strings.Index(summary, "Open:")
	if reviewIdx == -1 || openIdx == -1 {
		t.Fatalf("both sections should be present, got:\n%s", summary)
	}
	if reviewIdx > openIdx {
		t.Error("Review should appear first at same priority due to +5 boost")
	}
}

func TestBuildBoardSummaryHigherOpenBeatsLowerReview(t *testing.T) {
	// P2 OPEN (40) beats P3 REVIEW (35)
	open := []*ticket.Ticket{
		{ID: "st_open01", Title: "Open P2", Priority: ticket.PriorityP2, Status: ticket.StatusOpen},
		{ID: "st_open02", Title: "Open P4", Priority: ticket.PriorityP4, Status: ticket.StatusOpen},
	}
	review := []*ticket.Ticket{
		{ID: "st_rev01", Title: "Review P3", Priority: ticket.PriorityP3, Status: ticket.StatusReview},
	}

	summary := buildBoardSummary("proj", "", open, review)

	// Open section should come first since P2 OPEN (40) > P3 REVIEW (35)
	openIdx := strings.Index(summary, "Open:")
	reviewIdx := strings.Index(summary, "Review:")
	if openIdx == -1 || reviewIdx == -1 {
		t.Fatalf("both sections should be present, got:\n%s", summary)
	}
	if openIdx > reviewIdx {
		t.Error("Open section should appear before Review when open scores higher")
	}
	// All tickets from both batches should be present
	if !strings.Contains(summary, "st_open01") || !strings.Contains(summary, "st_open02") {
		t.Error("all open tickets should be shown")
	}
	if !strings.Contains(summary, "st_rev01") {
		t.Error("review tickets should also be shown")
	}
}

func TestBuildBoardSummaryReviewBeatsNextPriorityDown(t *testing.T) {
	// P3 REVIEW (35) beats P3 OPEN (30) — the +5 boost means REVIEW
	// beats the same priority level.
	open := []*ticket.Ticket{
		{ID: "st_open01", Title: "Open P3", Priority: ticket.PriorityP3, Status: ticket.StatusOpen},
		{ID: "st_open02", Title: "Open P4", Priority: ticket.PriorityP4, Status: ticket.StatusOpen},
	}
	review := []*ticket.Ticket{
		{ID: "st_rev01", Title: "Review P3", Priority: ticket.PriorityP3, Status: ticket.StatusReview},
		{ID: "st_rev02", Title: "Review P3", Priority: ticket.PriorityP3, Status: ticket.StatusReview},
	}

	summary := buildBoardSummary("proj", "", open, review)

	// Review should come first, and header should show total count
	reviewIdx := strings.Index(summary, "Review:")
	openIdx := strings.Index(summary, "Open:")
	if reviewIdx == -1 || openIdx == -1 {
		t.Fatalf("both sections should be present, got:\n%s", summary)
	}
	if reviewIdx > openIdx {
		t.Error("Review should appear first when P3 REVIEW (35) beats P3 OPEN (30)")
	}
	if !strings.Contains(summary, "4 tickets ready") {
		t.Errorf("header should show total ticket count (4), got:\n%s", summary)
	}
}

func TestBuildBoardSummaryDesignExample1(t *testing.T) {
	// From DESIGN.md example:
	// OPEN: P2, P3, P4  |  REVIEW: P3, P3
	// Highest = P2 OPEN (40) → Open section first, sorted: P2, P3, P4
	open := []*ticket.Ticket{
		{ID: "st_p3open", Title: "P3 Open", Priority: ticket.PriorityP3, Status: ticket.StatusOpen},
		{ID: "st_p2open", Title: "P2 Open", Priority: ticket.PriorityP2, Status: ticket.StatusOpen},
		{ID: "st_p4open", Title: "P4 Open", Priority: ticket.PriorityP4, Status: ticket.StatusOpen},
	}
	review := []*ticket.Ticket{
		{ID: "st_p3rev1", Title: "P3 Review 1", Priority: ticket.PriorityP3, Status: ticket.StatusReview},
		{ID: "st_p3rev2", Title: "P3 Review 2", Priority: ticket.PriorityP3, Status: ticket.StatusReview},
	}

	summary := buildBoardSummary("proj", "", open, review)

	if !strings.Contains(summary, "5 tickets ready") {
		t.Errorf("expected total 5 tickets, got:\n%s", summary)
	}

	// Open should come first
	openIdx := strings.Index(summary, "Open:")
	reviewIdx := strings.Index(summary, "Review:")
	if openIdx > reviewIdx {
		t.Error("Open section should appear before Review when open scores higher")
	}

	// Verify sort order within open section: P2 before P3 before P4
	p2Idx := strings.Index(summary, "st_p2open")
	p3Idx := strings.Index(summary, "st_p3open")
	p4Idx := strings.Index(summary, "st_p4open")
	if p2Idx > p3Idx || p3Idx > p4Idx {
		t.Errorf("tickets not sorted P2 < P3 < P4, got:\n%s", summary)
	}
}

func TestBuildBoardSummaryDesignExample2(t *testing.T) {
	// From DESIGN.md example:
	// OPEN: P3, P4  |  REVIEW: P2, P3
	// Highest = P2 REVIEW (45) → Review section first, sorted: P2, P3
	open := []*ticket.Ticket{
		{ID: "st_p3open", Title: "P3 Open", Priority: ticket.PriorityP3, Status: ticket.StatusOpen},
		{ID: "st_p4open", Title: "P4 Open", Priority: ticket.PriorityP4, Status: ticket.StatusOpen},
	}
	review := []*ticket.Ticket{
		{ID: "st_p3rev", Title: "P3 Review", Priority: ticket.PriorityP3, Status: ticket.StatusReview},
		{ID: "st_p2rev", Title: "P2 Review", Priority: ticket.PriorityP2, Status: ticket.StatusReview},
	}

	summary := buildBoardSummary("proj", "", open, review)

	if !strings.Contains(summary, "4 tickets ready") {
		t.Errorf("expected total 4 tickets, got:\n%s", summary)
	}

	// Review should come first
	reviewIdx := strings.Index(summary, "Review:")
	openIdx := strings.Index(summary, "Open:")
	if reviewIdx > openIdx {
		t.Error("Review section should appear before Open when review scores higher")
	}

	// Verify sort order within review section: P2 before P3
	p2Idx := strings.Index(summary, "st_p2rev")
	p3Idx := strings.Index(summary, "st_p3rev")
	if p2Idx > p3Idx {
		t.Errorf("tickets not sorted P2 < P3, got:\n%s", summary)
	}
}

func TestTicketScore(t *testing.T) {
	tests := []struct {
		name     string
		priority ticket.Priority
		status   ticket.Status
		want     int
	}{
		{"P0 OPEN", ticket.PriorityP0, ticket.StatusOpen, 60},
		{"P0 REVIEW", ticket.PriorityP0, ticket.StatusReview, 65},
		{"P3 OPEN", ticket.PriorityP3, ticket.StatusOpen, 30},
		{"P3 REVIEW", ticket.PriorityP3, ticket.StatusReview, 35},
		{"P5 OPEN", ticket.PriorityP5, ticket.StatusOpen, 10},
		{"P5 REVIEW", ticket.PriorityP5, ticket.StatusReview, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &ticket.Ticket{Priority: tt.priority, Status: tt.status}
			got := ticketScore(tk)
			if got != tt.want {
				t.Errorf("ticketScore() = %d, want %d", got, tt.want)
			}
		})
	}
}
