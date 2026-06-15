package telemetry

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/chaktihor/signalplane/internal/store"
)

type ReliableSink struct {
	primary store.TelemetrySink
	path    string
	mu      sync.Mutex
}

type replayEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func NewReliableSink(primary store.TelemetrySink, path string) *ReliableSink {
	return &ReliableSink{primary: primary, path: path}
}

func (sink *ReliableSink) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = sink.Replay()
		}
	}
}

func (sink *ReliableSink) WriteMetrics(metrics []store.Metric) error {
	err := sink.primary.WriteMetrics(metrics)
	if err != nil {
		sink.spool("metrics", metrics)
	}
	return err
}

func (sink *ReliableSink) WriteLogs(logs []store.Log) error {
	err := sink.primary.WriteLogs(logs)
	if err != nil {
		sink.spool("logs", logs)
	}
	return err
}

func (sink *ReliableSink) WriteTraces(traces []store.Trace) error {
	err := sink.primary.WriteTraces(traces)
	if err != nil {
		sink.spool("traces", traces)
	}
	return err
}

func (sink *ReliableSink) WriteUptimeResult(monitor store.UptimeMonitor) error {
	err := sink.primary.WriteUptimeResult(monitor)
	if err != nil {
		sink.spool("uptime", monitor)
	}
	return err
}

func (sink *ReliableSink) Replay() error {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	file, err := os.Open(sink.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	var failed []replayEvent
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var event replayEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if err := sink.replayEvent(event); err != nil {
			failed = append(failed, event)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return sink.rewrite(failed)
}

func (sink *ReliableSink) replayEvent(event replayEvent) error {
	switch event.Type {
	case "metrics":
		var metrics []store.Metric
		if err := json.Unmarshal(event.Payload, &metrics); err != nil {
			return nil
		}
		return sink.primary.WriteMetrics(metrics)
	case "logs":
		var logs []store.Log
		if err := json.Unmarshal(event.Payload, &logs); err != nil {
			return nil
		}
		return sink.primary.WriteLogs(logs)
	case "traces":
		var traces []store.Trace
		if err := json.Unmarshal(event.Payload, &traces); err != nil {
			return nil
		}
		return sink.primary.WriteTraces(traces)
	case "uptime":
		var monitor store.UptimeMonitor
		if err := json.Unmarshal(event.Payload, &monitor); err != nil {
			return nil
		}
		return sink.primary.WriteUptimeResult(monitor)
	default:
		return nil
	}
}

func (sink *ReliableSink) spool(eventType string, payload any) {
	if sink.path == "" {
		return
	}
	sink.mu.Lock()
	defer sink.mu.Unlock()
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	event, err := json.Marshal(replayEvent{Type: eventType, Payload: data})
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(sink.path), 0o755); err != nil {
		return
	}
	file, err := os.OpenFile(sink.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(event, '\n'))
}

func (sink *ReliableSink) rewrite(events []replayEvent) error {
	if len(events) == 0 {
		if err := os.Remove(sink.path); os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	tmp := sink.path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			_ = file.Close()
			return err
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, sink.path)
}
