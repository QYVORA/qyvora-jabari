// Package events implements the canonical QYVORA JSONL event stream. Every
// framework that participates in the shared contract emits the same event
// shape (schema_version, timestamp, execution_id, framework, level, event,
// data), one JSON object per line, so agents, CI pipelines and the future
// orchestrator can consume any framework's stream uniformly.
package events

import (
	"io"

	shared "github.com/QYVORA/qyvora-events"
)

// SchemaVersion is the event schema version every emitted event carries.
const SchemaVersion = shared.SchemaVersion

// Event names. The dotted namespace is stable across frameworks so an agent
// can key on scan.started / finding.discovered regardless of which framework
// produced the stream.
const (
	ScanStarted       = "scan.started"
	ScanCompleted     = "scan.completed"
	StageStarted      = "stage.started"
	StageCompleted    = "stage.completed"
	ModuleStarted     = "module.started"
	ModuleCompleted   = "module.completed"
	FindingDiscovered = "finding.discovered"
	EvidenceCollected = "evidence.collected"
	Warning           = "warning"
	Error             = "error"
	ReportGenerated   = "report.generated"
)

// Levels classify how an event should be treated by a consumer.
const (
	LevelInfo    = shared.LevelInfo
	LevelWarning = shared.LevelWarning
	LevelError   = shared.LevelError
)

type Event = shared.Event
type Emitter = shared.Emitter

// New returns an emitter writing JSONL to w.
func New(w io.Writer, execID string) *Emitter { return shared.New(w, execID) }
