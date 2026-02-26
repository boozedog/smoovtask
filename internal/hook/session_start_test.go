package hook

import (
	"strings"
	"testing"

	"github.com/boozedog/smoovbrain/internal/ticket"
)

func TestBuildBoardSummaryOpen(t *testing.T) {
	open := []*ticket.Ticket{
		{ID: "sb_a7Kx2m", Title: "Add rate limiting", Priority: ticket.PriorityP2},
		{ID: "sb_c1Dw4n", Title: "Fix CORS headers", Priority: ticket.PriorityP3},
	}

	summary := buildBoardSummary("api-server", open, nil)

	if !strings.Contains(summary, "smoovbrain — api-server — 2 OPEN tickets ready") {
		t.Errorf("missing header, got:\n%s", summary)
	}
	if !strings.Contains(summary, "sb_a7Kx2m") {
		t.Error("missing ticket ID sb_a7Kx2m")
	}
	if !strings.Contains(summary, "sb pick") {
		t.Error("missing pick instruction")
	}
}

func TestBuildBoardSummaryReview(t *testing.T) {
	review := []*ticket.Ticket{
		{ID: "sb_test01", Title: "Reviewed ticket", Priority: ticket.PriorityP2},
	}

	summary := buildBoardSummary("proj", nil, review)

	if !strings.Contains(summary, "REVIEW") {
		t.Error("should show REVIEW when review tickets exist")
	}
	if !strings.Contains(summary, "sb review") {
		t.Error("missing review instruction")
	}
}

func TestBuildBoardSummaryEmpty(t *testing.T) {
	summary := buildBoardSummary("proj", nil, nil)
	if summary != "" {
		t.Errorf("expected empty summary, got: %q", summary)
	}
}

func TestBuildBoardSummaryReviewPreferred(t *testing.T) {
	open := []*ticket.Ticket{
		{ID: "sb_open01", Title: "Open ticket", Priority: ticket.PriorityP3},
	}
	review := []*ticket.Ticket{
		{ID: "sb_rev01", Title: "Review ticket", Priority: ticket.PriorityP2},
	}

	summary := buildBoardSummary("proj", open, review)

	// Review should be preferred over open
	if !strings.Contains(summary, "REVIEW") {
		t.Error("review tickets should be preferred")
	}
	if strings.Contains(summary, "sb_open01") {
		t.Error("open tickets should not appear when review tickets exist")
	}
}
