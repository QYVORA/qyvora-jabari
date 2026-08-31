package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/QYVORA/qyvora-jabari/internal/banner"
)

// ANSI style codes. The brand accent is lime green (#85C236); hard errors are
// red, warnings Amber and neutral text white/dim (bettercap/msfconsole
// convention).
const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiAmber   = "\x1b[33m"
	ansiWhite   = "\x1b[37m"
	ansiOnBlack = "\x1b[40m"
	// Brand lime green #85C236 (RGB 133,194,54) — the primary logo accent.
	ansiLime = "\x1b[38;2;133;194;54m"
	// Brand deep blue-black #19222B (RGB 25,34,43) — the logo background,
	// used for the banner's negative space.
	ansiDeep = "\x1b[38;2;25;34;43m"
)

// consoleSectionWidth is the default layout width used by the console HUD.
const consoleSectionWidth = 60

// consoleUI renders styled output for the interactive console. Colors are
// enabled only when the writer is a terminal and NO_COLOR is not set, so
// piped/scripted output stays plain.
type consoleUI struct {
	w     io.Writer
	color bool
	// width is the live terminal column count used by the HUD; the console
	// refreshes it before every render.
	width int
}

// newConsoleUI builds a UI for w, auto-detecting color support.
func newConsoleUI(w io.Writer) *consoleUI {
	u := &consoleUI{w: w, width: consoleSectionWidth}
	if os.Getenv("NO_COLOR") == "" {
		u.color = writerIsTerminal(w)
	}
	return u
}

// Enabled reports whether colors are active.
func (u *consoleUI) Enabled() bool { return u.color }

// paint wraps s in code/Reset when colors are active.
func (u *consoleUI) paint(s, code string) string {
	if !u.color || s == "" {
		return s
	}
	return code + s + ansiReset
}

// Red paints a string red (hard errors and the prompt accent).
func (u *consoleUI) Red(s string) string { return u.paint(s, ansiRed) }

// Green paints a string in the brand lime green (#85C236) — success.
func (u *consoleUI) Green(s string) string { return u.paint(s, ansiLime) }

// Amber paints a string amber (warning).
func (u *consoleUI) Amber(s string) string { return u.paint(s, ansiAmber) }

// White paints a string plain white (information).
func (u *consoleUI) White(s string) string { return u.paint(s, ansiWhite) }

// BoldWhite paints a string bold white (headings).
func (u *consoleUI) BoldWhite(s string) string { return u.paint(s, ansiBold+ansiWhite) }

// DimWhite paints a string dim white (muted).
func (u *consoleUI) DimWhite(s string) string { return u.paint(s, ansiDim+ansiWhite) }

// Section prints a clean section header. The title is emphasized with the
// brand accent (uppercase) and whitespace rather than a full-width rule.
func (u *consoleUI) Section(title string) {
	label := strings.TrimSpace(title)
	if label == "" {
		_, _ = fmt.Fprintln(u.w)
		return
	}
	_, _ = fmt.Fprintf(u.w, "\n  %s\n", u.Green(strings.ToUpper(label)))
}

// Rule prints a soft section break — a blank line, never a rule line.
func (u *consoleUI) Rule() {
	_, _ = fmt.Fprintln(u.w)
}

// Clear clears the terminal screen (a no-op when not a terminal).
func (u *consoleUI) Clear() {
	if writerIsTerminal(u.w) {
		_, _ = fmt.Fprint(u.w, "\x1b[2J\x1b[H")
	}
}

// KV prints a "  key: value" pair with the key emphasized.
func (u *consoleUI) KV(key, value string) {
	_, _ = fmt.Fprintf(u.w, "  %s %s\n", u.BoldWhite(key+":"), u.White(value))
}

// Status prints a status line with a colored glyph (bettercap style):
// [+] success, [*] info, [!] warning, [x] error, [>] system, [-] neutral.
func (u *consoleUI) Status(glyph, format string, args ...any) {
	_, _ = fmt.Fprintf(u.w, "  %s %s\n", u.Glyph(glyph), u.White(fmt.Sprintf(format, args...)))
}

// Err prints a hard-error line with a red [x] glyph.
func (u *consoleUI) Err(format string, args ...any) {
	_, _ = fmt.Fprintf(u.w, "  %s %s\n", u.paint("[x]", ansiBold+ansiRed), u.Red(fmt.Sprintf(format, args...)))
}

// Glyph returns a colored "[x]" token for a status glyph character.
// Success uses the brand lime, the system prompt uses bold lime, warnings
// Amber, hard errors red, and neutral states dim white.
func (u *consoleUI) Glyph(glyph string) string {
	switch glyph {
	case "+":
		return u.paint("[+]", ansiLime)
	case "*":
		return u.paint("[*]", ansiWhite)
	case "!":
		return u.paint("[!]", ansiAmber)
	case "x", "X":
		return u.paint("[x]", ansiBold+ansiRed)
	case ">":
		return u.paint("[>]", ansiBold+ansiLime)
	case "-":
		return u.paint("[-]", ansiDim+ansiWhite)
	default:
		return u.paint("["+glyph+"]", ansiWhite)
	}
}

// Prompt builds the interactive prompt with the framework name in bold brand
// lime (the deliberate accent color) and a bold white chevron.
func (u *consoleUI) Prompt(name string) string {
	return u.paint(name, ansiBold+ansiLime) + u.paint(" > ", ansiBold+ansiWhite)
}

// HUD prints a one-line status bar with a brand lime edge block and space
// padding between left and right sections spanning the terminal width.
func (u *consoleUI) HUD(left, right string) {
	cols := u.width
	if cols < 20 {
		cols = 80
	}
	pad := cols - runeWidth(left) - runeWidth(right) - 1
	if pad < 1 {
		pad = 1
	}
	_, _ = fmt.Fprintf(u.w, "%s %s%s\n", u.paint("▮", ansiBold+ansiLime), left, strings.Repeat(" ", pad)+right)
}

// Table prints a header and aligned rows, padded to the widest visible cell.
func (u *consoleUI) Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = runeWidth(h)
	}
	for _, r := range rows {
		for i := 0; i < len(headers) && i < len(r); i++ {
			if l := runeWidth(r[i]); l > widths[i] {
				widths[i] = l
			}
		}
	}

	var b strings.Builder
	for i, h := range headers {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(padTo(u.BoldWhite(h), widths[i]))
	}
	_, _ = fmt.Fprintln(u.w, b.String())

	for _, r := range rows {
		var rb strings.Builder
		for i := 0; i < len(headers); i++ {
			if i > 0 {
				rb.WriteString("  ")
			}
			var cell string
			if i < len(r) {
				cell = r[i]
			}
			rb.WriteString(padTo(u.White(cell), widths[i]))
		}
		_, _ = fmt.Fprintln(u.w, rb.String())
	}
}

// Banner prints the canonical brand banner (internal/banner) followed by the
// tagline, mapping each glyph to the logo palette so the mark renders in the
// brand colors on a real terminal and stays plain when colors are off.
func (u *consoleUI) Banner(tagline string) {
	for _, line := range strings.Split(banner.Art, "\n") {
		var b strings.Builder
		for _, r := range line {
			b.WriteString(u.paint(string(r), bannerGlyphColor(r)))
		}
		_, _ = fmt.Fprintln(u.w, b.String())
	}
	_, _ = fmt.Fprintln(u.w)
	_, _ = fmt.Fprintln(u.w, u.BoldWhite(tagline))
	_, _ = fmt.Fprintln(u.w)
}

// bannerGlyphColor maps a glyph of the canonical brand art to the logo
// palette: '%' is the lime-green (#85C236) robot body, '#' the deep
// blue-black (#19222B) circular background, and '+'/'*' the white (#FFFFFF)
// structural details.
func bannerGlyphColor(r rune) string {
	switch r {
	case '%':
		return ansiLime
	case '#':
		return ansiDeep
	case '+', '*':
		return ansiWhite
	}
	return ansiWhite
}

// BannerFoot prints the version footer and a help hint under the banner.
func (u *consoleUI) BannerFoot(version string) {
	u.Status(">", "v %s", version)
	_, _ = fmt.Fprintln(u.w, u.DimWhite("type 'help' for commands, 'quit' to exit"))
	_, _ = fmt.Fprintln(u.w)
}

// spinner renders a live progress indicator on the current line while a
// command executes, so the operator always sees that work is in progress
// instead of a frozen screen followed by a fresh prompt. It draws with
// carriage-return + clear-line so it never leaves trailing junk, and every
// write is mutex-guarded because command output can arrive concurrently.
type spinner struct {
	mu     sync.Mutex
	w      io.Writer
	paint  func(string) string
	label  string
	active bool
	done   chan struct{}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// startSpinner launches a spinner that redraws on the current line. It
// returns nil when colors are disabled (plain/piped output), in which case
// callers skip spinner handling entirely.
func (u *consoleUI) startSpinner(w io.Writer, label string) *spinner {
	if !u.Enabled() {
		return nil
	}
	s := &spinner{
		w:      w,
		paint:  u.Green,
		label:  label,
		active: true,
		done:   make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *spinner) run() {
	i := 0
	for {
		select {
		case <-s.done:
			return
		default:
		}
		s.mu.Lock()
		if s.active {
			frame := spinnerFrames[i%len(spinnerFrames)]
			_, _ = fmt.Fprintf(s.w, "\r\x1b[K%s %s %s", s.paint("[*]"), s.label, frame)
		}
		s.mu.Unlock()
		i++
		time.Sleep(100 * time.Millisecond)
	}
}

// Stop halts the spinner and clears its line so the next prompt renders
// cleanly. It must be called exactly once per started spinner.
func (s *spinner) Stop() {
	close(s.done)
	s.mu.Lock()
	if s.active {
		_, _ = fmt.Fprintf(s.w, "\r\x1b[K")
		s.active = false
	}
	s.mu.Unlock()
}

// Pause clears the spinner line and stops redrawing so an interactive
// sub-prompt (for example the authorization confirmation) is not clobbered
// by concurrent output.
func (s *spinner) Pause() {
	s.mu.Lock()
	if s.active {
		_, _ = fmt.Fprintf(s.w, "\r\x1b[K")
		s.active = false
	}
	s.mu.Unlock()
}

// Resume restarts redrawing after Pause.
func (s *spinner) Resume() {
	s.mu.Lock()
	s.active = true
	s.mu.Unlock()
}

// busyLabel maps a console command line to a short loading message shown
// beside the spinner. The empty string marks commands that complete
// instantly, which get no spinner at all.
func busyLabel(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	switch strings.ToLower(fields[0]) {
	case "assess", "run":
		return "running assessment"
	case "discover":
		return "discovering target"
	case "enumerate":
		return "inventorying applications"
	case "analyze":
		return "evaluating rules"
	case "validate":
		return "validating findings"
	case "target":
		return "resolving target"
	case "report", "sessions":
		return "rendering report"
	default:
		return ""
	}
}

// runeWidth counts the display width of s, stripping ANSI codes first and
// counting wide (CJK/emoji) characters as two columns.
func runeWidth(s string) int {
	if strings.Contains(s, "\x1b") {
		s = stripANSI(s)
	}
	n := 0
	for _, r := range s {
		if isWideRune(r) {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// isWideRune reports whether r occupies two terminal columns. The ranges
// mirror the Unicode EastAsianWidth property used by wcwidth.
func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F,
		r >= 0x2329 && r <= 0x232A,
		r >= 0x2E80 && r <= 0xA4CF,
		r >= 0xAC00 && r <= 0xD7A3,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE10 && r <= 0xFE19,
		r >= 0xFE30 && r <= 0xFE6F,
		r >= 0xFF00 && r <= 0xFF60,
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1F64F,
		r >= 0x1F900 && r <= 0x1F9FF:
		return true
	}
	return false
}

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

// padTo pads s (which may contain ANSI codes) with trailing spaces to a
// visible width of n columns.
func padTo(s string, n int) string {
	pad := n - runeWidth(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

// writerIsTerminal reports whether w is an interactive character device.
func writerIsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
