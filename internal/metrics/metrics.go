// Package metrics provides opt-in runtime timing for shy.
// Enabled via SHY_METRICS=1; disabled path is intentionally cheap.
package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Span is one timed segment; the zero value is safe to End().
type Span struct {
	name  string
	start time.Time
}

type record struct {
	Name       string `json:"name"`
	StartNanos int64  `json:"start_nanos"`
	DurationNs int64  `json:"duration_ns"`
}

var fileLock sync.Mutex

// enabled reads the env var at call time so tests and operators can toggle live.
func enabled() bool {
	return os.Getenv("SHY_METRICS") == "1"
}

// Start begins a span; returns a zero-value Span when disabled.
func Start(name string) Span {
	if !enabled() {
		return Span{}
	}
	return Span{name: name, start: time.Now()}
}

// End records the span; safe to call on a zero-value receiver.
func (s Span) End() {
	if s.name == "" {
		return
	}
	r := record{
		Name:       s.name,
		StartNanos: s.start.UnixNano(),
		DurationNs: time.Since(s.start).Nanoseconds(),
	}
	_ = write(r) // metrics must never break shy
}

// write appends one record as JSONL to $HOME/.shy/metrics.jsonl.
func write(r record) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".shy", "metrics.jsonl")
	fileLock.Lock()
	defer fileLock.Unlock()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(r)
}
