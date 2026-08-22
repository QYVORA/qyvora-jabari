package events

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// parseLines decodes every JSON line from a stream for assertions.
func parseLines(t *testing.T, raw string) []Event {
	t.Helper()
	var out []Event
	for _, line := range strings.Split(strings.TrimSuffix(raw, "\n"), "\n") {
		if line == "" {
			continue
		}
		var e Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("invalid JSONL line %q: %v", line, err)
		}
		out = append(out, e)
	}
	return out
}

func TestEmitterWritesJSONL(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, "sess-1")
	em.Info("jabari", ScanStarted, map[string]any{"profile": "standard"})
	em.Fail("jabari", Error, map[string]any{"message": "boom"})

	events := parseLines(t, buf.String())
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	first := events[0]
	if first.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %q, want %q", first.SchemaVersion, SchemaVersion)
	}
	if first.ExecutionID != "sess-1" {
		t.Errorf("execution_id = %q, want sess-1", first.ExecutionID)
	}
	if first.Framework != "jabari" {
		t.Errorf("framework = %q, want jabari", first.Framework)
	}
	if first.Level != LevelInfo {
		t.Errorf("level = %q, want %q", first.Level, LevelInfo)
	}
	if first.Event != ScanStarted {
		t.Errorf("event = %q, want %q", first.Event, ScanStarted)
	}
	if first.Timestamp.IsZero() {
		t.Error("timestamp is zero")
	}
	if first.Data["profile"] != "standard" {
		t.Errorf("data.profile = %v, want standard", first.Data["profile"])
	}
	if events[1].Level != LevelError {
		t.Errorf("second event level = %q, want %q", events[1].Level, LevelError)
	}
}

func TestEmitterNilSafe(_ *testing.T) {
	var em *Emitter
	em.Info("jabari", ScanStarted, nil) // must not panic
}

func TestEmitterConcurrent(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, "sess-9")
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 50; j++ {
				em.Info("jabari", FindingDiscovered, map[string]any{"n": j})
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	events := parseLines(t, buf.String())
	if len(events) != 400 {
		t.Fatalf("got %d events, want 400", len(events))
	}
}

func TestEmitterMarshalNeverLosesFields(t *testing.T) {
	var buf bytes.Buffer
	em := New(&buf, "exec-42")
	em.Info("jabari", EvidenceCollected, map[string]any{
		"finding_id": "fin-1",
		"source":     "dumpsys",
		"count":      3,
	})
	raw := buf.String()
	for _, want := range []string{`"schema_version"`, `"execution_id"`, `"framework"`, `"event"`, `"evidence.collected"`} {
		if !strings.Contains(raw, want) {
			t.Errorf("stream missing %s in %q", want, raw)
		}
	}
}
