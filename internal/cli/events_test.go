package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenEventsWriterDisabled(t *testing.T) {
	w, closeFn, err := openEventsWriter("")
	if err != nil {
		t.Fatalf("openEventsWriter(\"\"): %v", err)
	}
	if w != nil || closeFn != nil {
		t.Error("disabled spec should yield nil writer/close, got non-nil")
	}
}

func TestOpenEventsWriterStdoutStderr(t *testing.T) {
	for _, spec := range []string{"stdout", "stderr"} {
		w, closeFn, err := openEventsWriter(spec)
		if err != nil {
			t.Fatalf("openEventsWriter(%q): %v", spec, err)
		}
		if w == nil {
			t.Errorf("openEventsWriter(%q) returned nil writer", spec)
		}
		if closeFn != nil {
			t.Errorf("console destinations must not return a close function")
		}
	}
}

func TestOpenEventsWriterFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	w, closeFn, err := openEventsWriter(path)
	if err != nil {
		t.Fatalf("openEventsWriter(file): %v", err)
	}
	if w == nil || closeFn == nil {
		t.Fatal("file destination must return a writer and close function")
	}
	if _, err := io.WriteString(w, "line\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := closeFn(); err != nil {
		t.Fatalf("close: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "line\n" {
		t.Errorf("file content = %q, want %q", string(data), "line\n")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("events file mode = %v, want 0600", info.Mode().Perm())
	}
}
