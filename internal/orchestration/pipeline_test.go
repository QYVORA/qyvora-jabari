package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/internal/events"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// fakeStage is a scripted pipeline stage.
type fakeStage struct {
	name string
	err  error
}

func (f *fakeStage) Name() string { return f.name }

func (f *fakeStage) Run(_ context.Context, _ *core.Env) error { return f.err }

func TestPipelineEmitsStageEvents(t *testing.T) {
	var buf bytes.Buffer
	env := &core.Env{
		Session: models.NewSession(),
		Events:  events.New(&buf, "exec-1"),
	}

	pipe := NewPipeline()
	pipe.Add(&fakeStage{name: "discovery"})
	pipe.Add(&fakeStage{name: "analysis"})
	if err := pipe.Run(context.Background(), env); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var started, completed int
	for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		var e events.Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("bad event line %q: %v", line, err)
		}
		switch e.Event {
		case events.StageStarted:
			started++
			if e.Level != events.LevelInfo {
				t.Errorf("stage.started level = %q, want info", e.Level)
			}
		case events.StageCompleted:
			completed++
		}
	}
	if started != 2 || completed != 2 {
		t.Errorf("got %d started / %d completed events, want 2/2\nstream:\n%s", started, completed, buf.String())
	}
}

func TestPipelineEmitsFailureEventAtErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	env := &core.Env{
		Session: models.NewSession(),
		Events:  events.New(&buf, "exec-2"),
	}

	pipe := NewPipeline()
	pipe.Add(&fakeStage{name: "discovery", err: errors.New("no adb")})
	if err := pipe.Run(context.Background(), env); err == nil {
		t.Fatal("Run should fail")
	}

	var sawError bool
	for _, line := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		var e events.Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		if e.Event == events.StageCompleted && e.Level == events.LevelError {
			sawError = true
			if e.Data["message"] == nil {
				t.Error("failure stage.completed event missing message")
			}
		}
	}
	if !sawError {
		t.Errorf("expected an error-level stage.completed event, stream:\n%s", buf.String())
	}
}

func TestPipelineWithoutEmitterIsSafe(t *testing.T) {
	env := &core.Env{Session: models.NewSession()}
	pipe := NewPipeline()
	pipe.Add(&fakeStage{name: "discovery"})
	if err := pipe.Run(context.Background(), env); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(env.Session.Stages) != 1 {
		t.Errorf("stages = %v, want [discovery]", env.Session.Stages)
	}
}
