package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-script/ast"
	"github.com/QYVORA/qyvora-script/binding"
	"github.com/QYVORA/qyvora-script/interpreter"
	"github.com/QYVORA/qyvora-script/parser"

	"github.com/QYVORA/qyvora-jabari/internal/analysis"
	"github.com/QYVORA/qyvora-jabari/internal/core"
	"github.com/QYVORA/qyvora-jabari/internal/discovery"
	"github.com/QYVORA/qyvora-jabari/internal/enumeration"
	errs "github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/internal/events"
	"github.com/QYVORA/qyvora-jabari/internal/orchestration"
	"github.com/QYVORA/qyvora-jabari/internal/poc"
	"github.com/QYVORA/qyvora-jabari/internal/reporting"
	"github.com/QYVORA/qyvora-jabari/internal/validation"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// allowShellOut gates the shared engine's exec/run.cmd construct. It is set by
// the persistent --allow-shellout flag; the binding also honours
// interpreter.AllowShellOutEnv.
var allowShellOut bool

// scriptCapability names one script-callable jabari module. jabari's
// internal/capabilities package is a detection list rather than an ID registry,
// so the binding derives stable IDs from the tool's existing tier/module
// surface (see root.go). No new capability is introduced: each ID maps to a
// command jabari already exposes.
type scriptCapability struct {
	ID   string
	Name string
	Tier string
}

// scriptCapabilities is the module surface this binding exposes to scripts.
func scriptCapabilities() []scriptCapability {
	return []scriptCapability{
		{"assess", "run the full assessment pipeline", "recon/exploit"},
		{"jabari.apk", "statically analyze an offline APK artifact", "recon"},
		{"jabari.discover", "identify the current target", "recon"},
		{"jabari.enumerate", "inventory applications on the current target", "recon"},
		{"jabari.analyze", "evaluate rules against the current target", "recon"},
		{"jabari.validate", "confirm detected findings on the current target", "exploit"},
		{"jabari.poc", "run proof-of-concept modules against the current target", "exploit"},
	}
}

// newScriptCmd adds the flat `script` utility that runs the shared
// qyvora-script engine (ARCH-004 Part C). `script run <file>` executes a
// script; `script build <file>` prints a dry-run plan. A bare `script <file>`
// runs the file for convenience.
func newScriptCmd() *cobra.Command {
	run := func(dryRun bool) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, args []string) error {
			return runScriptFile(cmd, args[0], dryRun)
		}
	}
	cmd := &cobra.Command{
		Use:   "script <file>",
		Short: "run a .qs script with the shared scripting engine",
		Args:  cobra.ExactArgs(1),
		RunE:  run(false),
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "run <file>",
			Short: "execute a script file",
			Args:  cobra.ExactArgs(1),
			RunE:  run(false),
		},
		&cobra.Command{
			Use:   "build <file>",
			Short: "validate a script file and print a dry-run plan",
			Args:  cobra.ExactArgs(1),
			RunE:  run(true),
		},
	)
	return cmd
}

// runScriptFile parses (build) or executes (run) a script against jabari's
// binding.
func runScriptFile(cmd *cobra.Command, path string, dryRun bool) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return errs.WrapExitError(1, "reading script", err)
	}
	prog, err := parser.Parse(string(src))
	if err != nil {
		return errs.NewExitError(2, err.Error())
	}
	if dryRun {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "script plan "+path)
		for _, line := range ast.Describe(prog) {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "  -", line)
		}
		return nil
	}
	b := &scriptBinding{
		ctx:   cmd.Context(),
		out:   cmd.OutOrStdout(),
		allow: allowShellOut,
		cfg:   map[string]string{},
	}
	engine := interpreter.NewEngine(b)
	engine.Events = os.Stderr
	if err := engine.RunProgram(prog); err != nil {
		return errs.NewExitError(2, err.Error())
	}
	return nil
}

// scriptBinding implements binding.Binding over jabari's assessment pipeline.
// jabari is one-shot: a module call runs the pipeline against the configured
// target and keeps the finished session for $(...) interpolation and reports.
type scriptBinding struct {
	ctx   context.Context
	out   io.Writer
	allow bool
	cfg   map[string]string
	last  *models.Session
	lastT *models.Target
	lastA int
}

var _ binding.Binding = (*scriptBinding)(nil)

// Capabilities lists the capability IDs this binding accepts.
func (b *scriptBinding) Capabilities() []string {
	caps := scriptCapabilities()
	ids := make([]string, 0, len(caps))
	for _, c := range caps {
		ids = append(ids, c.ID)
	}
	return ids
}

// AllowShellOut reports whether exec/run.cmd may run: only with the explicit
// flag or the shared opt-in environment variable.
func (b *scriptBinding) AllowShellOut() bool { return b.allow || shellOutEnv() }

func shellOutEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(interpreter.AllowShellOutEnv))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// Call dispatches one engine capability invocation.
func (b *scriptBinding) Call(capabilityID string, args []binding.Value) (binding.Value, error) {
	switch capabilityID {
	case binding.CapEcho:
		_, _ = fmt.Fprintln(b.out, asStr(args, 0))
		return nil, nil

	case binding.CapSetConfig:
		b.cfg[asStr(args, 0)] = asStr(args, 1)
		return nil, nil

	case binding.CapGetConfig:
		return b.cfg[asStr(args, 0)], nil

	case binding.CapStart:
		return nil, b.start(asStr(args, 0))

	case binding.CapStop, binding.CapIsRunning:
		if capabilityID == binding.CapIsRunning {
			return false, nil
		}
		return nil, nil

	case binding.CapShow:
		return nil, b.show(asStr(args, 0))

	case binding.CapReport:
		return nil, b.report(asStr(args, 0))

	case binding.CapProp:
		return b.prop(asStr(args, 0)), nil

	case binding.CapExec:
		return nil, b.exec(asStr(args, 0))
	}
	return nil, fmt.Errorf("unknown capability %q", capabilityID)
}

// get returns the first configured value among keys.
func (b *scriptBinding) get(keys ...string) string {
	for _, k := range keys {
		if v, ok := b.cfg[k]; ok {
			return v
		}
	}
	return ""
}

// target builds and authorizes a target from the script's config map.
func (b *scriptBinding) target() (*models.Target, error) {
	apk := b.get("assess.apk", "apk")
	usb := b.get("assess.usb", "usb")
	ip := b.get("assess.ip", "ip")
	t := &models.Target{ID: models.NewID("tgt"), CreatedAt: nowUTC()}
	switch {
	case apk != "":
		t.Name = "apk " + filepath.Base(apk)
		t.Type = models.TargetAPK
		t.Address = apk
	case usb != "":
		t.Name = "USB device " + usb
		t.Type = models.TargetUSB
		t.Serial = usb
	case ip != "":
		t.Name = "network device " + ip
		t.Type = models.TargetNetwork
		t.Address = ip
	default:
		return nil, fmt.Errorf("no target configured; set assess.apk, assess.usb or assess.ip")
	}
	// The script author configured the target explicitly, so authorization is
	// recorded the same way --authorized/-y records it on the CLI.
	t.Auth = granted(t)
	return t, nil
}

// profile resolves the assessment profile from the script config and config
// file, defaulting to standard.
func (b *scriptBinding) profile() (orchestration.Profile, error) {
	name := b.get("assess.profile", "profile")
	if name == "" {
		name = cfg.GetString("profile")
	}
	if name == "" {
		name = string(orchestration.ProfileStandard)
	}
	if !orchestration.IsValid(name) {
		return "", errs.NewExitError(2, "unknown profile "+name)
	}
	return orchestration.Profile(name), nil
}

// start runs the pipeline associated with a module id and keeps the session.
func (b *scriptBinding) start(id string) error {
	t, err := b.target()
	if err != nil {
		return err
	}
	profile, err := b.profile()
	if err != nil {
		return err
	}
	if err := targets.Set(t); err != nil {
		return errs.NewExitError(2, "setting target: "+err.Error())
	}
	sess, err := b.run(t, profile, id)
	if err != nil {
		return err
	}
	b.last = sess
	b.lastT = t
	return nil
}

// run executes the selected pipeline against t and returns the finished
// session. It mirrors runPipeline but takes an explicit target and module id so
// the binding can also run a single stage.
func (b *scriptBinding) run(t *models.Target, profile orchestration.Profile, id string) (*models.Session, error) {
	highRisk := isTrue(b.get("assess.poc-high-risk", "poc-high-risk"))
	pipe, err := selectPipeline(id, t, profile, highRisk)
	if err != nil {
		return nil, err
	}
	env, cleanup, err := newAssessmentEnv(b.ctx, t, profile)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	emitter := env.Events
	if emitter != nil {
		emitter.Info("jabari", events.ScanStarted, map[string]any{
			"profile": string(profile), "target_id": t.ID, "target_type": string(t.Type),
		})
	}
	if err := pipe.Run(b.ctx, env); err != nil {
		if emitter != nil {
			emitter.Fail("jabari", events.Error, map[string]any{"message": err.Error()})
		}
		return nil, err
	}
	env.Session.Finish()
	if path, err := persistSession(env.Session); err == nil {
		log.Info("session saved to %s", path)
		if emitter != nil {
			emitter.Info("jabari", events.ReportGenerated, map[string]any{"path": path, "format": "json"})
		}
	}
	if emitter != nil {
		emitter.Info("jabari", events.ScanCompleted, map[string]any{
			"findings": len(env.Session.Findings), "risk_score": env.Session.RiskScore,
			"risk_level": env.Session.RiskLevel, "stages": env.Session.Stages,
		})
	}
	b.lastA = len(env.Apps)
	return env.Session, nil
}

// selectPipeline maps a module id to the pipeline it runs.
func selectPipeline(id string, t *models.Target, profile orchestration.Profile, highRisk bool) (*orchestration.Pipeline, error) {
	switch id {
	case "", "assess":
		var pipe *orchestration.Pipeline
		if t.Type == models.TargetAPK {
			pipe = orchestration.ForAPKProfile(profile)
		} else {
			pipe = orchestration.ForProfile(profile)
		}
		if highRisk {
			pipe.Add(&poc.Stage{AllowHighRisk: true})
		}
		return pipe, nil
	case "jabari.apk":
		return orchestration.ForAPKProfile(profile), nil
	case "jabari.discover":
		return single(&discovery.Stage{}), nil
	case "jabari.enumerate":
		return single(&enumeration.Stage{}), nil
	case "jabari.analyze":
		return single(&analysis.Stage{}), nil
	case "jabari.validate":
		return single(&validation.Stage{}), nil
	case "jabari.poc":
		return single(&poc.Stage{AllowHighRisk: highRisk}), nil
	default:
		return nil, fmt.Errorf("unknown module %q", id)
	}
}

func single(s core.Stage) *orchestration.Pipeline {
	pipe := orchestration.NewPipeline()
	pipe.Add(s)
	return pipe
}

// show prints metadata for a module id.
func (b *scriptBinding) show(id string) error {
	for _, c := range scriptCapabilities() {
		if c.ID == id {
			printer.PrintTable([]string{"capability", "name", "tier"}, [][]string{
				{c.ID, c.Name, c.Tier},
			})
			return nil
		}
	}
	return fmt.Errorf("unknown module %q", id)
}

// report writes a Markdown report artifact for the most recent run.
func (b *scriptBinding) report(path string) error {
	if b.last == nil {
		return fmt.Errorf("no assessment to report; run a module first")
	}
	dir := path
	if dir == "" {
		dir = reportDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file := filepath.Join(dir, "report-"+b.last.ID+".md")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := &reporting.Writer{Format: reporting.FormatMarkdown, Out: f}
	if err := w.Render(b.ctx, b.last); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(b.out, "report written to "+file)
	return nil
}

// prop resolves live properties for $(...) interpolation from the last run.
func (b *scriptBinding) prop(name string) binding.Value {
	if b.last == nil {
		return nil
	}
	switch name {
	case "session.id", "result.id":
		return b.last.ID
	case "result.profile":
		return b.last.Profile
	case "result.apps":
		return fmt.Sprintf("%d", b.lastA)
	case "result.findings":
		return fmt.Sprintf("%d", len(b.last.Findings))
	case "result.stages":
		return strings.Join(b.last.Stages, ",")
	case "result.score":
		return fmt.Sprintf("%d", b.last.RiskScore)
	case "result.level":
		return b.last.RiskLevel
	case "target.name":
		if b.lastT != nil {
			return b.lastT.DisplayName()
		}
	}
	return nil
}

// exec runs a one-shot jabari command line. The interpreter only reaches here
// after AllowShellOut reported true.
func (b *scriptBinding) exec(line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	code := ExecuteArgsContext(b.ctx, strings.Fields(line))
	if code != 0 {
		return fmt.Errorf("command exited with code %d", code)
	}
	return nil
}

func asStr(args []binding.Value, i int) string {
	if i >= len(args) || args[i] == nil {
		return ""
	}
	if s, ok := args[i].(string); ok {
		return s
	}
	return fmt.Sprintf("%v", args[i])
}

func isTrue(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
