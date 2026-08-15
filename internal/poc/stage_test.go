package poc

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	errs "github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/internal/events"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

func TestStageRequiresAuthorization(t *testing.T) {
	env := testEnv(newFakeTransport())
	env.Target.Auth.Granted = false
	env.Session.Findings = []*models.Finding{debuggableFinding()}

	err := (&Stage{}).Run(context.Background(), env)
	if err == nil {
		t.Fatal("poc stage must refuse an unauthorized target")
	}
	var exitErr *errs.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 3 {
		t.Fatalf("error = %v, want exit code 3", err)
	}
}

func TestStageRecordsRunsAndExploitability(t *testing.T) {
	ft := newFakeTransport()
	ft.canned["run-as com.debug.app id"] = models.Response{
		Stdout: []byte("uid=10048(u0_a48)\n"), ExitCode: 0,
	}
	env := testEnv(ft)
	f := debuggableFinding()
	env.Session.Findings = []*models.Finding{f}

	var buf bytes.Buffer
	env.Events = events.New(&buf, "exec-1")

	s := &Stage{}
	if err := s.Run(context.Background(), env); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// The proven run is recorded on the session.
	run := findPocRun(env.Session.Pocs, runAsModuleID)
	if run == nil {
		t.Fatalf("no poc run recorded for %s (have %d)", runAsModuleID, len(env.Session.Pocs))
	}
	if run.Status != models.PocProven {
		t.Errorf("run status = %s, want proven", run.Status)
	}
	if run.FindingID != f.ID || len(run.Evidence) == 0 {
		t.Errorf("run = %+v", run)
	}

	// The finding's exploitability ladder advanced to exploited.
	if f.Exploitability != "exploited" {
		t.Errorf("exploitability = %q, want exploited", f.Exploitability)
	}

	// module lifecycle events were streamed.
	out := buf.String()
	if !strings.Contains(out, `"event":"module.completed"`) || !strings.Contains(out, `"status":"proven"`) {
		t.Errorf("event stream missing module.completed/proven: %s", out)
	}
}

func TestStageNotProvenMarksDynamic(t *testing.T) {
	env := testEnv(newFakeTransport())
	f := debuggableFinding()
	env.Session.Findings = []*models.Finding{f}

	if err := (&Stage{}).Run(context.Background(), env); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if f.Exploitability != "dynamic" {
		t.Errorf("exploitability = %q, want dynamic", f.Exploitability)
	}
}

func TestStageNoTransportIsNoop(t *testing.T) {
	env := testEnv(newFakeTransport())
	env.Transport = nil
	env.Session.Findings = []*models.Finding{debuggableFinding()}

	if err := (&Stage{}).Run(context.Background(), env); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(env.Session.Pocs) != 0 {
		t.Errorf("poc runs recorded without a transport: %d", len(env.Session.Pocs))
	}
}

func TestStageHighRiskSkippedByDefault(t *testing.T) {
	env := testEnv(newFakeTransport())
	f := debuggableFinding()
	env.Session.Findings = []*models.Finding{f}

	if err := (&Stage{}).Run(context.Background(), env); err != nil {
		t.Fatalf("Run: %v", err)
	}
	run := findPocRun(env.Session.Pocs, exportedModuleID)
	if run == nil {
		t.Fatalf("no run recorded for %s", exportedModuleID)
	}
	if run.Status != models.PocSkipped {
		t.Errorf("status = %s, want skipped", run.Status)
	}
}

// findPocRun locates the recorded run for a module, or nil.
func findPocRun(runs []*models.PocRun, module string) *models.PocRun {
	for _, r := range runs {
		if r != nil && r.Module == module {
			return r
		}
	}
	return nil
}
