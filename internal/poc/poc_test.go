package poc

import (
	"context"
	"strings"
	"testing"

	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// fakeTransport is a scriptable transport that answers commands the way a
// real adb transport would, letting the PoC modules be exercised offline.
type fakeTransport struct {
	// canned maps a command prefix to the response to return.
	canned map[string]models.Response
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{canned: map[string]models.Response{}}
}

func (f *fakeTransport) Connect(_ context.Context) error { return nil }
func (f *fakeTransport) Disconnect() error               { return nil }
func (f *fakeTransport) String() string                  { return "fake" }
func (f *fakeTransport) Info(_ context.Context) (*models.DeviceInfo, error) {
	return &models.DeviceInfo{}, nil
}

func (f *fakeTransport) Execute(_ context.Context, req models.Request) (models.Response, error) {
	key := strings.Join(req.Args, " ")
	for prefix, resp := range f.canned {
		if strings.HasPrefix(key, prefix) {
			return resp, nil
		}
	}
	return models.Response{ExitCode: 1, Stderr: []byte("permission denied")}, nil
}

func testEnv(ft *fakeTransport) *core.Env {
	sess := models.NewSession()
	app := models.Application{
		PackageName: "com.debug.app",
		Debuggable:  true,
		Activities:  []string{".MainActivity", ".SecretActivity"},
	}
	sess.Apps = []models.Application{app}
	return &core.Env{
		Target:    &models.Target{Auth: models.Authorization{Granted: true}},
		Transport: ft,
		Session:   sess,
		Apps:      sess.Apps,
	}
}

func debuggableFinding() *models.Finding {
	return &models.Finding{
		ID:         "fnd-debug",
		RuleID:     "AND-007",
		Category:   "application-security",
		Status:     models.StatusDetected,
		Attributes: map[string]string{"package": "com.debug.app"},
		Evidence:   []models.Evidence{{Kind: models.KindConfiguration, Source: "ro.debuggable", Content: "1"}},
	}
}

func TestEngineProvesRunAsDebuggable(t *testing.T) {
	ft := newFakeTransport()
	ft.canned["run-as com.debug.app id"] = models.Response{
		Stdout: []byte("uid=10048(u0_a48) gid=10048(u0_a48)\n"), ExitCode: 0,
	}
	env := testEnv(ft)
	env.Session.Findings = []*models.Finding{debuggableFinding()}

	results := (&Engine{}).Run(context.Background(), env)
	res := moduleResult(results, runAsModuleID)
	if res == nil {
		t.Fatalf("no %s result in %+v", runAsModuleID, results)
	}
	if res.Status != models.PocProven {
		t.Fatalf("result = %+v, want proven", res)
	}
	if len(res.Evidence) == 0 || !strings.Contains(res.Evidence[0], "uid=10048") {
		t.Errorf("evidence = %v", res.Evidence)
	}
}

func TestEngineRunAsNotProvenWhenDenied(t *testing.T) {
	env := testEnv(newFakeTransport())
	env.Session.Findings = []*models.Finding{debuggableFinding()}

	results := (&Engine{}).Run(context.Background(), env)
	res := moduleResult(results, runAsModuleID)
	if res == nil {
		t.Fatalf("no %s result in %+v", runAsModuleID, results)
	}
	if res.Status != models.PocNotProven {
		t.Fatalf("status = %s, want not-proven", res.Status)
	}
}

// moduleResult finds the result produced by the named module, or nil.
func moduleResult(results []Result, module string) *Result {
	for i := range results {
		if results[i].Module == module {
			return &results[i]
		}
	}
	return nil
}

func TestEngineWorldReadableProven(t *testing.T) {
	ft := newFakeTransport()
	ft.canned["ls -l /data/data/com.debug.app"] = models.Response{
		Stdout: []byte("drwxrwxrwx u0_a48 u0_a48 com.debug.app\n"), ExitCode: 0,
	}
	env := testEnv(ft)
	f := &models.Finding{
		ID:         "fnd-wr",
		Category:   "application-security",
		Status:     models.StatusDetected,
		Attributes: map[string]string{"package": "com.debug.app"},
	}
	env.Session.Findings = []*models.Finding{f}

	results := (&Engine{}).Run(context.Background(), env)
	found := false
	for _, r := range results {
		if r.Module == worldReadableModuleID {
			found = true
			if r.Status != models.PocProven {
				t.Fatalf("world-readable status = %s, want proven", r.Status)
			}
		}
	}
	if !found {
		t.Fatal("world_readable_data module did not run")
	}
}

func TestEngineExportedActivityGatedByHighRisk(t *testing.T) {
	ft := newFakeTransport()
	ft.canned["am start -n com.debug.app/.MainActivity"] = models.Response{
		Stdout: []byte("Starting: Intent { cmp=com.debug.app/.MainActivity }\n"), ExitCode: 0,
	}
	ft.canned["pidof com.debug.app"] = models.Response{Stdout: []byte("1234\n"), ExitCode: 0}
	env := testEnv(ft)
	env.Session.Findings = []*models.Finding{debuggableFinding()}

	// Without opt-in the high-risk module is skipped.
	results := (&Engine{}).Run(context.Background(), env)
	skipped := false
	for _, r := range results {
		if r.Module == exportedModuleID {
			skipped = true
			if r.Status != models.PocSkipped {
				t.Fatalf("status = %s, want skipped without opt-in", r.Status)
			}
		}
	}
	if !skipped {
		t.Fatal("exported_activity module was not considered")
	}

	// With opt-in the module proves the activity is launchable.
	results = (&Engine{AllowHighRisk: true}).Run(context.Background(), env)
	proven := false
	for _, r := range results {
		if r.Module == exportedModuleID {
			proven = true
			if r.Status != models.PocProven {
				t.Fatalf("status = %s, want proven with opt-in", r.Status)
			}
			if len(r.Evidence) == 0 {
				t.Error("proven exported activity missing evidence")
			}
		}
	}
	if !proven {
		t.Fatal("exported_activity module did not prove with opt-in")
	}
}

func TestEngineModuleFilterRestrictsModules(t *testing.T) {
	ft := newFakeTransport()
	ft.canned["run-as com.debug.app id"] = models.Response{
		Stdout: []byte("uid=10048(u0_a48)\n"), ExitCode: 0,
	}
	env := testEnv(ft)
	env.Session.Findings = []*models.Finding{debuggableFinding()}

	eng := &Engine{AllowHighRisk: true, ModuleFilter: map[string]bool{exportedModuleID: true}}
	results := eng.Run(context.Background(), env)
	for _, r := range results {
		if r.Module != exportedModuleID {
			t.Errorf("filter leaked module %s", r.Module)
		}
	}
	if len(results) == 0 {
		t.Fatal("filtered run produced no results")
	}
}

func TestEngineSkipsResolvedFindings(t *testing.T) {
	ft := newFakeTransport()
	ft.canned["run-as com.debug.app id"] = models.Response{
		Stdout: []byte("uid=10048(u0_a48)\n"), ExitCode: 0,
	}
	env := testEnv(ft)
	f := debuggableFinding()
	f.Status = models.StatusResolved
	env.Session.Findings = []*models.Finding{f}

	if results := (&Engine{}).Run(context.Background(), env); len(results) != 0 {
		t.Fatalf("resolved finding produced %d results, want 0", len(results))
	}
}

func TestEngineNoTransportNoResults(t *testing.T) {
	env := testEnv(newFakeTransport())
	env.Transport = nil
	env.Session.Findings = []*models.Finding{debuggableFinding()}
	if results := (&Engine{}).Run(context.Background(), env); len(results) != 0 {
		t.Fatalf("no-transport run produced %d results", len(results))
	}
}

func TestComponentName(t *testing.T) {
	cases := []struct{ pkg, activity, want string }{
		{"com.x", "MainActivity", "com.x/.MainActivity"},
		{"com.x", ".MainActivity", "com.x/.MainActivity"},
		{"com.x", "com.x.MainActivity", "com.x/com.x.MainActivity"},
		{"com.x", "com.x/.MainActivity", "com.x/.MainActivity"},
		{"com.x", "", "com.x/."},
	}
	for _, tc := range cases {
		if got := componentName(tc.pkg, tc.activity); got != tc.want {
			t.Errorf("componentName(%q, %q) = %q, want %q", tc.pkg, tc.activity, got, tc.want)
		}
	}
}

func TestCandidateAppsOrderingAndDedup(t *testing.T) {
	env := testEnv(newFakeTransport())
	env.Apps = []models.Application{
		{PackageName: "com.b"},
		{PackageName: "com.debug.app", Debuggable: true},
		{PackageName: "com.a"},
	}
	f := &models.Finding{Attributes: map[string]string{"package": "com.a"}}
	got := candidateApps(env, f)
	want := []string{"com.a", "com.debug.app", "com.b"}
	if len(got) != len(want) {
		t.Fatalf("candidateApps = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidateApps = %v, want %v", got, want)
		}
	}
}
