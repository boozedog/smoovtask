package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Events handles the SSE endpoint for streaming events to the browser.
func (h *Handler) Events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := h.broker.Subscribe()
	defer h.broker.Unsubscribe(ch)

	// Send initial keepalive.
	fmt.Fprintf(w, ": keepalive\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			// Send as a "refresh" event that htmx SSE extension listens for.
			fmt.Fprintf(w, "event: refresh\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}
