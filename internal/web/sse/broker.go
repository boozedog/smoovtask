package sse

import (
	"log/slog"
	"sync"

	"github.com/boozedog/smoovtask/internal/event"
)

// Broker manages SSE client connections and broadcasts events.
type Broker struct {
	mu      sync.RWMutex
	clients map[chan event.Event]struct{}
}

// NewBroker creates a new SSE broker.
func NewBroker() *Broker {
	return &Broker{
		clients: make(map[chan event.Event]struct{}),
	}
}

// Subscribe registers a new client and returns its event channel.
// The caller must call Unsubscribe when done.
func (b *Broker) Subscribe() chan event.Event {
	ch := make(chan event.Event, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	slog.Debug("sse client connected", "total", b.Count())
	return ch
}

// Unsubscribe removes a client and closes its channel.
func (b *Broker) Unsubscribe(ch chan event.Event) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
	slog.Debug("sse client disconnected", "total", b.Count())
}

// Broadcast sends an event to all connected clients.
// Slow clients that can't keep up will have their event dropped.
func (b *Broker) Broadcast(e event.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- e:
		default:
			slog.Warn("dropping event for slow sse client")
		}
	}
}

// Count returns the number of connected clients.
func (b *Broker) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}
