package sse

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"os"
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
	offsets := make(map[string]int64)

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
			workChanged, activityChanged := classifyDirtyFiles(dirty, offsets)
			clear(dirty)
			if workChanged {
				w.broker.Broadcast(event.Event{Event: "refresh-work"})
			}
			if activityChanged {
				w.broker.Broadcast(event.Event{Event: "refresh-activity"})
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("fsnotify error", "err", err)
		}
	}
}

func classifyDirtyFiles(dirty map[string]struct{}, offsets map[string]int64) (workChanged, activityChanged bool) {
	for path := range dirty {
		currentOffset := offsets[path]
		fileInfo, err := os.Stat(path)
		if err != nil {
			continue
		}
		if fileInfo.Size() < currentOffset {
			currentOffset = 0
		}

		f, err := os.Open(path)
		if err != nil {
			continue
		}

		if _, err := f.Seek(currentOffset, io.SeekStart); err != nil {
			_ = f.Close()
			continue
		}

		reader := bufio.NewReader(f)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) > 0 && line[len(line)-1] == '\n' {
				currentOffset += int64(len(line))
				line = line[:len(line)-1]
				if len(line) == 0 {
					if err == io.EOF {
						break
					}
					continue
				}

				var ev event.Event
				if json.Unmarshal(line, &ev) == nil {
					activityChanged = true
					if strings.HasPrefix(ev.Event, "ticket.") || strings.HasPrefix(ev.Event, "status.") {
						workChanged = true
					}
				}
			}

			if err != nil {
				if err == io.EOF {
					break
				}
				break
			}
		}

		offsets[path] = currentOffset
		_ = f.Close()
	}

	return workChanged, activityChanged
}
