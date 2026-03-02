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

type agentPing struct {
	runID  string
	hook   string
	ticket string
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
				for _, ping := range pings {
					data := map[string]any{"hook": ping.hook}
					if ping.ticket != "" {
						data["ticket"] = ping.ticket
					}
					w.broker.Broadcast(event.Event{
						Event: "agent-ping",
						RunID: ping.runID,
						Data:  data,
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
// classifies them. Agent pings are returned in order, deduplicated by run ID.
func readNewLines(path string, offsets map[string]int64) (workChanged, activityChanged bool, agentPings []agentPing) {
	runIDToPing := make(map[string]agentPing)
	runIDOrder := make([]string, 0)

	currentOffset := offsets[path]
	fileInfo, err := os.Stat(path)
	if err != nil {
		return workChanged, activityChanged, agentPings
	}
	if fileInfo.Size() < currentOffset {
		currentOffset = 0
	}

	f, err := os.Open(path)
	if err != nil {
		return workChanged, activityChanged, agentPings
	}
	defer f.Close()

	if _, err := f.Seek(currentOffset, io.SeekStart); err != nil {
		return workChanged, activityChanged, agentPings
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
					if _, exists := runIDToPing[ev.RunID]; !exists {
						runIDOrder = append(runIDOrder, ev.RunID)
					}
					runIDToPing[ev.RunID] = agentPing{runID: ev.RunID, hook: ev.Event, ticket: ev.Ticket}
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
	for _, runID := range runIDOrder {
		agentPings = append(agentPings, runIDToPing[runID])
	}
	return workChanged, activityChanged, agentPings
}
