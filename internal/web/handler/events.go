package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Events handles the SSE endpoint for streaming events to the browser.
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.broker.Subscribe()
	defer h.broker.Unsubscribe(ch)

	// Send initial keepalive.
	if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}

	// Heartbeat detects stale connections when no events are flowing.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.Event == "agent-ping" {
				continue
			}
			eventName := ev.Event
			if eventName == "" {
				eventName = "refresh-activity"
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, data); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}

// AgentEvents handles SSE pings for agent activity indicators.
// If {runID} path param is present, only matching run pings are streamed.
// Otherwise, all agent pings are streamed with run_id in the data payload.
func (h *Handler) AgentEvents(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runID")

	rc := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.broker.Subscribe()
	defer h.broker.Unsubscribe(ch)

	if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.Event != "agent-ping" {
				continue
			}
			if runID != "" && ev.RunID != runID {
				continue
			}
			payload := map[string]string{"run_id": ev.RunID}
			if hook, ok := ev.Data["hook"].(string); ok {
				payload["hook"] = hook
			}
			data, err := json.Marshal(payload)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: ping\ndata: %s\n\n", data); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
