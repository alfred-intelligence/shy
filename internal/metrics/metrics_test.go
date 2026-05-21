package metrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Zero-value Span must End() without panic.
func TestSpanZeroValueIsNoop(t *testing.T) {
	var s Span
	s.End()
}

// Disabled by default: no metrics file created.
func TestDisabledByDefault(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SHY_METRICS", "")

	Start("test").End()

	if _, err := os.Stat(filepath.Join(tmp, ".shy", "metrics.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("metrics file created when disabled: %v", err)
	}
}

// Enabled: span appends one JSONL record containing the span name.
func TestEnabledAppendsJSONL(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SHY_METRICS", "1")

	if err := os.MkdirAll(filepath.Join(tmp, ".shy"), 0o700); err != nil {
		t.Fatal(err)
	}
	Start("install").End()

	data, err := os.ReadFile(filepath.Join(tmp, ".shy", "metrics.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"name":"install"`) {
		t.Fatalf("missing name in output: %s", data)
	}
}

// Multiple spans append independently.
func TestEnabledMultipleSpans(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("SHY_METRICS", "1")

	if err := os.MkdirAll(filepath.Join(tmp, ".shy"), 0o700); err != nil {
		t.Fatal(err)
	}
	Start("a").End()
	Start("b").End()
	Start("c").End()

	data, err := os.ReadFile(filepath.Join(tmp, ".shy", "metrics.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), "\n"); got != 3 {
		t.Fatalf("expected 3 records, got %d", got)
	}
}

// Disabled path must stay cheap; flag regressions in CI.
func BenchmarkStartEndDisabled(b *testing.B) {
	b.Setenv("SHY_METRICS", "")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Start("bench").End()
	}
}
