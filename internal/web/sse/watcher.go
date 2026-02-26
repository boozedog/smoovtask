package sse

import (
	"log/slog"
	"strings"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/fsnotify/fsnotify"
)

// Watcher watches the events directory for new JSONL lines and feeds them to a broker.
type Watcher struct {
	dir     string
	broker  *Broker
	watcher *fsnotify.Watcher
}

// NewWatcher creates and starts a file watcher on the events directory.
func NewWatcher(eventsDir string, broker *Broker) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		dir:     eventsDir,
		broker:  broker,
		watcher: fw,
	}

	if err := fw.Add(eventsDir); err != nil {
		_ = fw.Close()
		return nil, err
	}

	go w.loop()
	return w, nil
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

func (w *Watcher) loop() {
	// Debounce: collect file change notifications and flush at most once per second.
	const debounce = time.Second
	timer := time.NewTimer(debounce)
	timer.Stop()
	dirty := make(map[string]struct{})

	for {
		select {
		case ev, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(ev.Name, ".jsonl") {
				continue
			}
			if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) {
				if len(dirty) == 0 {
					timer.Reset(debounce)
				}
				dirty[ev.Name] = struct{}{}
			}
		case <-timer.C:
			clear(dirty)
			// Send a single refresh signal regardless of how many events arrived.
			w.broker.Broadcast(event.Event{Event: "refresh"})
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("fsnotify error", "err", err)
		}
	}
}
