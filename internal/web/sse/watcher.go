package sse

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/boozedog/smoovtask/internal/event"
	"github.com/fsnotify/fsnotify"
)

// Watcher watches the events directory for new JSONL lines and feeds them to a broker.
type Watcher struct {
	dir     string
	broker  *Broker
	watcher *fsnotify.Watcher

	mu      sync.Mutex
	offsets map[string]int64 // file path → last read offset
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
		offsets: make(map[string]int64),
	}

	// Snapshot current file sizes so we only stream new events.
	w.snapshotOffsets()

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

// snapshotOffsets records current file sizes for all existing JSONL files.
func (w *Watcher) snapshotOffsets() {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(w.dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		w.offsets[path] = info.Size()
	}
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
			for path := range dirty {
				w.consumeNewLines(path)
			}
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

// consumeNewLines advances the offset past any new lines without broadcasting individually.
func (w *Watcher) consumeNewLines(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	info, err := os.Stat(path)
	if err != nil {
		return
	}
	w.offsets[path] = info.Size()
}
