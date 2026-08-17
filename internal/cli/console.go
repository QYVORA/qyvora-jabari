package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"

	errs "github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/internal/orchestration"
	"github.com/QYVORA/qyvora-jabari/internal/reporting"
	"github.com/QYVORA/qyvora-jabari/internal/version"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
)

// jabariConsole is the interactive, Metasploit-style console. Running bare
// "jabari" drops the operator into it; every one-shot CLI command remains
// available as a console command so nothing is lost in the REPL.
type jabariConsole struct {
	ctx     context.Context
	out     io.Writer
	ui      *consoleUI
	rl      *readline.Instance
	spin    *spinner
	history []string
	// cwd is the console's working directory, used by host shell commands
	// (!command, shell, cd, pwd) and device shell. It persists across
	// commands so the operator can navigate while working.
	cwd string
}

// runConsole launches the interactive console. When stdin is not a terminal
// (pipes, scripts, CI) it degrades to plain line-by-line reading so command
// sequences can be fed in non-interactively.
func runConsole(ctx context.Context) error {
	c := &jabariConsole{ctx: ctx, out: os.Stdout, ui: newConsoleUI(os.Stdout)}
	if !writerIsTerminal(os.Stdin) {
		return c.runPlain()
	}
	rl, err := readline.NewEx(&readline.Config{
		Prompt:       c.ui.Prompt("jabari"),
		HistoryFile:  c.historyPath(),
		AutoComplete: readline.NewPrefixCompleter(c.completer()...),
	})
	if err != nil {
		// The line editor could not start; degrade to plain reading.
		fmt.Fprintf(c.out, "line editing unavailable (%v); continuing in plain mode\n", err)
		return c.runPlain()
	}
	c.rl = rl
	defer func() { _ = rl.Close() }()

	c.ui.Banner("Android Security Assessment Framework")
	c.ui.BannerFoot(version.String())
	c.hud()
	c.ui.Status("*", "console ready. type 'help' for commands.")

	for {
		line, err := rl.Readline()
		if err != nil {
			return nil // EOF (Ctrl-D)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		c.history = append(c.history, line)
		quit, e := c.execWithPrompt(line)
		c.hud()
		if e != nil {
			c.ui.Err("%v", e)
		} else if quit {
			return nil
		}
	}
}

// runPlain reads one command per line from stdin without line editing.
func (c *jabariConsole) runPlain() error {
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		c.history = append(c.history, line)
		quit, err := c.exec(line)
		if err != nil {
			fmt.Fprintln(c.out, err)
		}
		if quit {
			return nil
		}
	}
	return sc.Err()
}

// historyPath returns the console history file, creating its parent
// directory (~/.qyvora) so line-editing history survives sessions.
func (c *jabariConsole) historyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".jabari_history"
	}
	dir := filepath.Join(home, ".qyvora")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ".jabari_history"
	}
	return filepath.Join(dir, "jabari_history")
}

// execWithPrompt runs one command while showing a live loading spinner, so
// the console never jumps straight from a long command back to a prompt
// without indicating work is in progress. The readline prompt is left alone
// and the spinner draws on the current line; when the command finishes the
// spinner clears itself and the next Readline renders the prompt fresh.
//
// The previous implementation refreshed the readline prompt from a ticker
// goroutine while commands ran. That raced with readline's own rendering and
// corrupted interactive sub-prompts (a typed "y" could be lost or misread,
// aborting authorization), so the ticker is gone entirely: command output and
// the spinner are the only writers during execution.
func (c *jabariConsole) execWithPrompt(line string) (bool, error) {
	if c.rl == nil {
		return c.exec(line)
	}
	// Interactive shell sessions own the terminal; a spinner would fight the
	// live child process, so any shell invocation runs without one.
	if c.shellKind(line) != "" {
		return c.exec(line)
	}
	c.spin = c.ui.startSpinner(c.out, busyLabel(line))
	defer func() {
		if c.spin != nil {
			c.spin.Stop()
			c.spin = nil
		}
	}()
	quit, err := c.exec(line)
	return quit, err
}

// pauseSpinner pauses the active loading spinner so an interactive read (the
// authorization confirmation) is not clobbered by concurrent redraws.
func (c *jabariConsole) pauseSpinner() {
	if c.spin != nil {
		c.spin.Pause()
	}
}

// resumeSpinner restarts the loading spinner after an interactive read.
func (c *jabariConsole) resumeSpinner() {
	if c.spin != nil {
		c.spin.Resume()
	}
}

// exec dispatches a single console command line.
func (c *jabariConsole) exec(line string) (bool, error) {
	fields := strings.Fields(line)
	if strings.HasPrefix(fields[0], "!") {
		// Shell escape hatch (bettercap convention): "!ls -l" runs "ls -l"
		// on the host, fully outside the command table.
		return false, c.runHostCommand(fields[0][1:], fields[1:])
	}
	cmd := strings.ToLower(fields[0])
	args := fields[1:]

	switch cmd {
	case "help", "?":
		c.help()
	case "banner":
		c.ui.Banner("Android Security Assessment Framework")
	case "version":
		c.printVersion()
	case "clear":
		c.ui.Clear()
	case "history":
		c.printHistory()
	case "quit", "exit", "bye":
		return true, nil
	case "shell":
		return false, c.runHostShell()
	case "device":
		return false, c.cmdDevice(args)
	case "cd":
		if len(args) == 0 {
			return false, c.changeDir("")
		}
		if len(args) == 1 && args[0] == "--prompt" {
			return c.changeDirOrEscape()
		}
		return false, c.changeDir(strings.Join(args, " "))
	case "pwd":
		c.printCwd()
	case "tools":
		c.cmdTools()
	case "capabilities", "caps":
		c.cmdCapabilities()
	case "target":
		return false, c.cmdTarget(args)
	case "assess", "run":
		return false, c.cmdAssess(args)
	case "discover":
		return false, runDiscover(c.ctx)
	case "enumerate":
		return false, runEnumerate(c.ctx)
	case "analyze":
		return false, runAnalyze(c.ctx)
	case "validate":
		return false, runValidate(c.ctx)
	case "poc":
		return false, c.cmdPoc(args)
	case "report", "sessions":
		return false, c.cmdReport(args)
	case "set":
		return false, c.cmdSet(args)
	case "get":
		return false, c.cmdGet(args)
	case "config":
		c.cmdConfig(args)
	default:
		return false, fmt.Errorf("unknown command %q (try 'help')", cmd)
	}
	return false, nil
}

// help prints the command catalogue grouped by theme.
func (c *jabariConsole) help() {
	type grp struct {
		name string
		cmds [][2]string
	}
	groups := []grp{
		{"Core", [][2]string{
			{"help", "show this command list"},
			{"banner", "print the jabari banner"},
			{"version", "print version information"},
			{"history", "show command history"},
			{"clear", "clear the screen"},
			{"quit", "leave the console"},
		}},
		{"Shell", [][2]string{
			{"!<command>", "run a host command (e.g. !ls -l)"},
			{"shell", "drop into an interactive host shell"},
			{"cd [dir]", "change the console working directory"},
			{"pwd", "print the working directory"},
			{"tools", "report the Android assessment toolchain"},
		}},
		{"Targets", [][2]string{
			{"target usb [serial]", "select a connected authorized device"},
			{"target ip <addr>", "select an authorized device by IP"},
			{"target show", "show the current target"},
			{"target list", "list known targets"},
			{"device shell", "open an interactive shell on the current target"},
			{"device shell <cmd>", "run one command on the current target"},
		}},
		{"Assessment", [][2]string{
			{"assess usb [serial]", "assess a connected authorized device"},
			{"assess ip <addr>", "assess one specific authorized device"},
			{"assess [--profile <p>]", "run the full pipeline on the current target"},
			{"discover", "identify the current target"},
			{"enumerate", "inventory applications on the current target"},
			{"analyze", "evaluate rules against the current target"},
			{"validate", "confirm detected findings on the current target"},
			{"poc [--poc-high-risk]", "run proof-of-concept modules on the authorized target"},
		}},
		{"Reporting", [][2]string{
			{"report [session-id]", "render a saved assessment report"},
			{"report list", "list saved assessment sessions"},
			{"sessions", "alias for 'report list'"},
		}},
		{"Configuration", [][2]string{
			{"set <key> <value>", "set profile, report.dir, report.format or timeout"},
			{"get <key>", "show a config value"},
			{"config", "dump the current configuration"},
		}},
	}
	for _, g := range groups {
		c.ui.Section(g.name)
		rows := make([][]string, 0, len(g.cmds))
		for _, item := range g.cmds {
			rows = append(rows, []string{item[0], item[1]})
		}
		c.ui.Table([]string{"command", "description"}, rows)
	}
}

// printVersion prints the build version, commit, date and build user.
func (c *jabariConsole) printVersion() {
	info := version.GetInfo()
	c.ui.Status(">", "jabari %s", info.Version)
	c.ui.KV("commit", info.Commit)
	c.ui.KV("built", info.Date)
	c.ui.KV("build user", info.BuildUser)
}

// printHistory prints the current session's command history.
func (c *jabariConsole) printHistory() {
	if len(c.history) == 0 {
		c.ui.Status("-", "no commands yet")
		return
	}
	for i, h := range c.history {
		fmt.Fprintf(c.out, "%4d  %s\n", i+1, h)
	}
}

// hud renders the persistent one-line status strip above the prompt: the
// current target on the left, profile and version on the right.
func (c *jabariConsole) hud() {
	if !c.ui.Enabled() {
		return
	}
	if w := readline.GetScreenWidth(); w > 20 {
		c.ui.width = w
	}

	var left strings.Builder
	left.WriteString(c.ui.DimWhite("target "))
	t := targets.Current()
	if t == nil {
		left.WriteString(c.ui.DimWhite("none"))
	} else {
		left.WriteString(c.ui.White(t.DisplayName()))
		if t.Authorized() {
			left.WriteString(" ")
			left.WriteString(c.ui.Green("authorized"))
		} else {
			left.WriteString(" ")
			left.WriteString(c.ui.Amber("not authorized"))
		}
	}

	profile := cfg.GetString("profile")
	if profile == "" {
		profile = string(orchestration.ProfileStandard)
	}
	var right strings.Builder
	right.WriteString(c.ui.DimWhite("profile "))
	right.WriteString(c.ui.White(profile))
	right.WriteString(c.ui.DimWhite("  ·  v "))
	right.WriteString(c.ui.White(version.String()))

	c.ui.HUD(left.String(), right.String())
}

// cmdTarget implements the "target" console command.
func (c *jabariConsole) cmdTarget(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: target usb [serial] | target ip <addr> | target show | target list")
	}
	switch strings.ToLower(args[0]) {
	case "usb":
		t, err := c.buildUSBTarget(args[1:])
		if err != nil {
			return err
		}
		return c.selectTarget(t)
	case "ip":
		if len(args) < 2 {
			return errors.New("usage: target ip <addr>")
		}
		t, err := c.buildIPTarget(args[1])
		if err != nil {
			return err
		}
		return c.selectTarget(t)
	case "show":
		t := targets.Current()
		if t == nil {
			return errs.NewExitError(2, "no target selected")
		}
		status := "not authorized"
		if t.Authorized() {
			status = "authorized"
		}
		c.ui.KV("target", t.DisplayName())
		c.ui.KV("id", t.ID)
		c.ui.KV("type", string(t.Type))
		c.ui.KV("status", status)
		if t.Device != nil {
			c.ui.KV("device", t.Device.Summary())
		}
		return nil
	case "list":
		list := targets.List()
		if len(list) == 0 {
			c.ui.Status("-", "no targets selected")
			return nil
		}
		rows := make([][]string, 0, len(list))
		for _, t := range list {
			status := c.ui.Amber("not authorized")
			if t.Authorized() {
				status = c.ui.Green("authorized")
			}
			rows = append(rows, []string{t.ID, string(t.Type), t.DisplayName(), status})
		}
		c.ui.Table([]string{"id", "type", "target", "status"}, rows)
		return nil
	default:
		return fmt.Errorf("unknown target subcommand %q", args[0])
	}
}

// buildUSBTarget resolves a device serial and builds an unauthorized USB
// target (mirrors "jabari target usb" / "jabari assess usb").
func (c *jabariConsole) buildUSBTarget(args []string) (*models.Target, error) {
	serial, err := resolveUSBSerial(c.ctx, args)
	if err != nil {
		return nil, err
	}
	return &models.Target{
		ID:        models.NewID("tgt"),
		Name:      "USB device " + serial,
		Type:      models.TargetUSB,
		Serial:    serial,
		CreatedAt: nowUTC(),
	}, nil
}

// buildIPTarget validates an address and builds an unauthorized network target.
func (c *jabariConsole) buildIPTarget(addr string) (*models.Target, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" || !strings.Contains(addr, ".") {
		return nil, errs.NewExitError(2, "invalid target address: "+addr)
	}
	return &models.Target{
		ID:        models.NewID("tgt"),
		Name:      "network device " + addr,
		Type:      models.TargetNetwork,
		Address:   addr,
		CreatedAt: nowUTC(),
	}, nil
}

// selectTarget prompts for authorization (when required) and makes the target
// current, mirroring the one-shot target commands.
func (c *jabariConsole) selectTarget(t *models.Target) error {
	if err := c.confirmAuth(t); err != nil {
		return err
	}
	if err := targets.Set(t); err != nil {
		return errs.NewExitError(2, "setting target: "+err.Error())
	}
	c.ui.Status("+", "target set: %s", t.DisplayName())
	return nil
}

// confirmAuth prompts for explicit authorization via the console prompt. On
// confirmation the target is granted (mirroring the one-shot authorize
// helper) so targets.Set and the pipeline's requireTarget accept it, and the
// shared authorization flag is set so later confirms are skipped.
func (c *jabariConsole) confirmAuth(t *models.Target) error {
	if authorizationFlags.authorized || cfg.GetBool("authorized") || strings.EqualFold(os.Getenv("QYVORA_AUTHORIZED"), "true") {
		t.Auth = granted(t)
		return nil
	}
	if !c.ui.Enabled() || c.rl == nil {
		return errs.NewExitError(3,
			"target authorization required; set authorized=true in config or run 'jabari assess --authorized' non-interactively")
	}
	// Pause the loading spinner so it cannot redraw over this interactive
	// prompt, then resume once the answer is read. The readline prompt is
	// swapped for the confirmation text so the typed answer lands on the
	// [y/N] line instead of a bare "jabari >" prompt.
	c.pauseSpinner()
	fmt.Fprintf(c.out, "\n%s\n", c.ui.BoldWhite("Authorized Android security assessment"))
	fmt.Fprintf(c.out, "  %s %s (%s)\n", c.ui.BoldWhite("Target:"), c.ui.White(t.DisplayName()), t.Type)
	fmt.Fprintf(c.out, "  %s %s\n", c.ui.BoldWhite("Scope:"), c.ui.DimWhite("authorized assessment only, scoped to this target"))
	c.rl.SetPrompt("  Confirm authorization? [y/N] ")
	answer, err := c.rl.Readline()
	c.rl.SetPrompt(c.ui.Prompt("jabari"))
	c.resumeSpinner()
	if err != nil {
		return err
	}
	if !strings.EqualFold(strings.TrimSpace(answer), "y") {
		return errors.New("authorization declined; assessment aborted")
	}
	t.Auth = granted(t)
	authorizationFlags.authorized = true
	return nil
}

// cmdAssess implements the "assess" / "run" console command.
func (c *jabariConsole) cmdAssess(args []string) error {
	profile := c.cfgProfile()
	t, err := c.assessTarget(args, &profile)
	if err != nil {
		return err
	}
	if t != nil {
		if err := c.confirmAuth(t); err != nil {
			return err
		}
		if err := targets.Set(t); err != nil {
			return errs.NewExitError(2, "setting target: "+err.Error())
		}
	}

	session, err := runPipeline(c.ctx, orchestration.Profile(profile))
	if err != nil {
		return err
	}
	return renderSession(c.ctx, session)
}

// assessTarget parses the assess arguments and returns the target to assess.
// A nil target (assess on the current target) is returned when no explicit
// usb/ip subcommand is given. The --profile flag is consumed from args.
func (c *jabariConsole) assessTarget(args []string, profile *string) (*models.Target, error) {
	// Parse a leading --profile <name> flag.
	if len(args) >= 2 && args[0] == "--profile" {
		*profile = args[1]
		args = args[2:]
	}
	if len(args) == 0 {
		if err := c.requireTarget(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	switch strings.ToLower(args[0]) {
	case "usb":
		return c.buildUSBTarget(args[1:])
	case "ip":
		if len(args) < 2 {
			return nil, errors.New("usage: assess ip <addr>")
		}
		if len(args) > 2 {
			return nil, fmt.Errorf("usage: assess ip <addr> — extra arguments ignored, got %q", strings.Join(args[1:], " "))
		}
		return c.buildIPTarget(args[1])
	default:
		return nil, fmt.Errorf("unknown assess target %q (try 'assess usb' or 'assess ip <addr>')", args[0])
	}
}

// cmdPoc implements the "poc" console command. It runs the PoC stage against
// the current authorized target; the stage itself enforces authorization and
// returns exit code 3 when the gate is not satisfied.
func (c *jabariConsole) cmdPoc(args []string) error {
	pocFlags.highRisk = false
	pocFlags.moduleFilter = nil
	for _, a := range args {
		switch {
		case a == "--poc-high-risk":
			pocFlags.highRisk = true
		case strings.HasPrefix(a, "--poc-module="):
			pocFlags.moduleFilter = append(pocFlags.moduleFilter,
				strings.Split(strings.TrimPrefix(a, "--poc-module="), ",")...)
		default:
			return errs.NewExitError(2, "unexpected argument "+a+" (see 'help')")
		}
	}
	return runPoc(c.ctx)
}

// requireTarget mirrors the one-shot requireTarget helper for the console.
func (c *jabariConsole) requireTarget() error {
	t := targets.Current()
	if t == nil {
		return errs.NewExitError(2, "no target selected; use 'target usb' or 'target ip <addr>' first")
	}
	if !t.Authorized() {
		return errs.NewExitError(2, "current target is not authorized: "+t.DisplayName())
	}
	return nil
}

// cfgProfile returns the configured assessment profile (with the standard
// default applied) from the console-managed configuration.
func (c *jabariConsole) cfgProfile() string {
	name := cfg.GetString("profile")
	if name == "" {
		name = string(orchestration.ProfileStandard)
	}
	return name
}

// cmdReport implements the "report" / "sessions" console command.
func (c *jabariConsole) cmdReport(args []string) error {
	if len(args) > 0 && (strings.EqualFold(args[0], "list") || args[0] == "--list") {
		return listSessions()
	}
	if len(args) > 0 && args[0] == "--format" && len(args) > 1 {
		reportFlags.format = args[1]
		args = args[2:]
	}
	path, err := sessionPath(args)
	if err != nil {
		return err
	}
	session, err := loadSession(path)
	if err != nil {
		return errs.NewExitError(1, "loading session: "+err.Error())
	}
	return renderSession(c.ctx, session)
}

// consoleConfigKeys are the values settable from the console.
var consoleConfigKeys = map[string]string{
	"profile":       "assessment profile (quick, standard, deep, application, device, network, compliance, research)",
	"report.dir":    "report output directory",
	"report.format": "report format (terminal, json, markdown, html)",
	"timeout":       "default timeout for device operations (e.g. 30s)",
}

// cmdSet implements "set <key> <value>".
func (c *jabariConsole) cmdSet(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: set <key> <value> (keys: profile, report.dir, report.format, timeout)")
	}
	key := strings.ToLower(args[0])
	value := strings.Join(args[1:], " ")
	switch key {
	case "profile":
		if !orchestration.IsValid(value) {
			return fmt.Errorf("unknown profile %q (quick, standard, deep, application, device, network, compliance, research)", value)
		}
		cfg.Set("profile", value)
	case "report.dir":
		cfg.Set("report.dir", value)
	case "report.format":
		if _, err := reporting.ParseFormat(value); err != nil {
			return errs.NewExitError(2, err.Error())
		}
		cfg.Set("report.format", value)
	case "timeout":
		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 {
			return fmt.Errorf("timeout: bad duration %q (use e.g. 30s)", value)
		}
		timeout = d
	default:
		return fmt.Errorf("unknown config key %q (keys: profile, report.dir, report.format, timeout)", key)
	}
	c.ui.Status("+", "%s = %s", key, value)
	return nil
}

// cmdGet implements "get <key>".
func (c *jabariConsole) cmdGet(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: get <key> (keys: profile, report.dir, report.format, timeout)")
	}
	key := strings.ToLower(args[0])
	switch key {
	case "profile":
		c.ui.KV(key, c.cfgProfile())
	case "report.dir":
		c.ui.KV(key, cfg.GetString("report.dir"))
	case "report.format":
		c.ui.KV(key, cfg.GetString("report.format"))
	case "timeout":
		c.ui.KV(key, timeout.String())
	default:
		return fmt.Errorf("unknown config key %q (keys: profile, report.dir, report.format, timeout)", key)
	}
	return nil
}

// cmdConfig dumps the current configuration.
func (c *jabariConsole) cmdConfig(args []string) {
	c.ui.Section("configuration")
	rows := [][]string{
		{"profile", c.cfgProfile()},
		{"report.dir", cfg.GetString("report.dir")},
		{"report.format", cfg.GetString("report.format")},
		{"timeout", timeout.String()},
	}
	c.ui.Table([]string{"key", "value"}, rows)
}

// completer builds the tab-completion tree.
func (c *jabariConsole) completer() []readline.PrefixCompleterInterface {
	item := func(name string, kids ...readline.PrefixCompleterInterface) readline.PrefixCompleterInterface {
		return readline.PcItem(name, kids...)
	}
	return []readline.PrefixCompleterInterface{
		item("help"),
		item("banner"),
		item("version"),
		item("clear"),
		item("history"),
		item("quit"),
		item("exit"),
		item("shell"),
		item("cd"),
		item("pwd"),
		item("tools"),
		item("device",
			item("shell"),
		),
		item("target",
			item("usb"),
			item("ip"),
			item("show"),
			item("list"),
		),
		item("assess",
			item("usb"),
			item("ip"),
			item("--profile",
				item("quick"), item("standard"), item("deep"), item("application"),
				item("device"), item("network"), item("compliance"), item("research"),
			),
		),
		item("run"),
		item("discover"),
		item("enumerate"),
		item("analyze"),
		item("validate"),
		item("report",
			item("list"),
			item("--format", item("terminal"), item("json"), item("markdown"), item("html")),
		),
		item("sessions"),
		item("set",
			item("profile", item("quick"), item("standard"), item("deep"), item("application"),
				item("device"), item("network"), item("compliance"), item("research")),
			item("report.dir"),
			item("report.format", item("terminal"), item("json"), item("markdown"), item("html")),
			item("timeout"),
		),
		item("get",
			item("profile"),
			item("report.dir"),
			item("report.format"),
			item("timeout"),
		),
		item("config"),
	}
}
