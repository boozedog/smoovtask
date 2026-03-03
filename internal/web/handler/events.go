package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Events handles the unified SSE endpoint for streaming all events to the
// browser. Both work/activity refreshes and agent-ping events are multiplexed
// over this single connection to avoid exhausting the browser's per-origin
// HTTP/1.1 connection limit (~6), which previously caused "clicks stop
// working" when multiple tabs were open.
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

			var eventName string
			var data []byte
			var err error

			if ev.Event == "agent-ping" {
				// Format agent pings with a compact payload matching
				// what the client-side JS expects.
				eventName = "ping"
				payload := map[string]string{"run_id": ev.RunID}
				if hook, ok := ev.Data["hook"].(string); ok {
					payload["hook"] = hook
				}
				if ticketID, ok := ev.Data["ticket"].(string); ok && ticketID != "" {
					payload["ticket"] = ticketID
				}
				data, err = json.Marshal(payload)
			} else {
				eventName = ev.Event
				if eventName == "" {
					eventName = "refresh-activity"
				}
				data, err = json.Marshal(ev)
			}

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
