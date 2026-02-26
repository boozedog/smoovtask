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
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: refresh\ndata: %s\n\n", data); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
