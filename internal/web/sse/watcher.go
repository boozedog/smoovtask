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
	// Agent pings are broadcast immediately on each file event.
	// Heavier refresh-work / refresh-activity events are debounced.
	const debounce = time.Second
	timer := time.NewTimer(debounce)
	timer.Stop()
	offsets := make(map[string]int64)
	var pendingWork, pendingActivity, timerRunning bool

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
				work, activity, pings := readNewLines(ev.Name, offsets)

				// Broadcast agent pings immediately — no debounce.
				for runID, hookName := range pings {
					w.broker.Broadcast(event.Event{
						Event: "agent-ping",
						RunID: runID,
						Data:  map[string]any{"hook": hookName},
					})
				}

				// Accumulate refresh flags for the debounced batch.
				pendingWork = pendingWork || work
				pendingActivity = pendingActivity || activity
				if (pendingWork || pendingActivity) && !timerRunning {
					timer.Reset(debounce)
					timerRunning = true
				}
			}
		case <-timer.C:
			timerRunning = false
			if pendingWork {
				w.broker.Broadcast(event.Event{Event: "refresh-work"})
				pendingWork = false
			}
			if pendingActivity {
				w.broker.Broadcast(event.Event{Event: "refresh-activity"})
				pendingActivity = false
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("fsnotify error", "err", err)
		}
	}
}

// readNewLines reads new JSONL lines from path (past the tracked offset) and
// classifies them. Agent-ping run IDs are returned in a deduplicated set.
func readNewLines(path string, offsets map[string]int64) (workChanged, activityChanged bool, agentPingRunIDs map[string]string) {
	agentPingRunIDs = make(map[string]string)

	currentOffset := offsets[path]
	fileInfo, err := os.Stat(path)
	if err != nil {
		return workChanged, activityChanged, agentPingRunIDs
	}
	if fileInfo.Size() < currentOffset {
		currentOffset = 0
	}

	f, err := os.Open(path)
	if err != nil {
		return workChanged, activityChanged, agentPingRunIDs
	}
	defer f.Close()

	if _, err := f.Seek(currentOffset, io.SeekStart); err != nil {
		return workChanged, activityChanged, agentPingRunIDs
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
				if strings.HasPrefix(ev.Event, "hook.") && ev.RunID != "" {
					agentPingRunIDs[ev.RunID] = ev.Event
				}
				if strings.HasPrefix(ev.Event, "ticket.") || strings.HasPrefix(ev.Event, "status.") {
					workChanged = true
				}
			}
		}

		if err != nil {
			break
		}
	}

	offsets[path] = currentOffset
	return workChanged, activityChanged, agentPingRunIDs
}
