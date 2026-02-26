package sse

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
		fw.Close()
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
				w.readNewLines(ev.Name)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("fsnotify error", "err", err)
		}
	}
}

// readNewLines reads any new lines appended to the file since last read.
func (w *Watcher) readNewLines(path string) {
	w.mu.Lock()
	offset := w.offsets[path]
	w.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		slog.Error("open jsonl", "path", path, "err", err)
		return
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, 0); err != nil {
			slog.Error("seek jsonl", "path", path, "err", err)
			return
		}
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e event.Event
		if err := json.Unmarshal(line, &e); err != nil {
			slog.Warn("malformed jsonl line", "err", err)
			continue
		}
		w.broker.Broadcast(e)
	}

	// Update offset to current position.
	pos, err := f.Seek(0, 1) // current position
	if err == nil {
		w.mu.Lock()
		w.offsets[path] = pos
		w.mu.Unlock()
	}
}
