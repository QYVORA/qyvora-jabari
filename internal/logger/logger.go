package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

type Level int

const (
	// LevelSilent suppresses all output.
	LevelSilent Level = iota
	LevelError
	LevelWarn
	LevelInfo
	LevelDebug
	LevelTrace
)

var levelNames = map[Level]string{
	LevelError: "ERROR",
	LevelWarn:  "WARN",
	LevelInfo:  "INFO",
	LevelDebug: "DEBUG",
	LevelTrace: "TRACE",
}

// ParseLevel converts a case-insensitive level name into a Level. Unknown
// names default to LevelInfo so configuration typos degrade to a usable log.
func ParseLevel(s string) Level {
	switch s {
	case "silent":
		return LevelSilent
	case "error":
		return LevelError
	case "warn":
		return LevelWarn
	case "debug":
		return LevelDebug
	case "trace":
		return LevelTrace
	default:
		return LevelInfo
	}
}

// String returns the canonical name for a level.
func (l Level) String() string {
	return levelNames[l]
}

type Logger struct {
	level   Level
	verbose bool
	quiet   bool
	writer  io.Writer
}

func New() *Logger {
	return &Logger{
		level:   LevelInfo,
		verbose: false,
		quiet:   false,
		writer:  os.Stderr,
	}
}

func (l *Logger) SetVerbose(v bool)     { l.verbose = v }
func (l *Logger) SetQuiet(q bool)       { l.quiet = q }
func (l *Logger) SetLevel(level Level)  { l.level = level }
func (l *Logger) SetWriter(w io.Writer) { l.writer = w }

func (l *Logger) log(level Level, format string, args ...any) {
	if l.quiet && level < LevelError {
		return
	}
	// verbose implies at least debug-level output, regardless of the
	// configured level.
	effective := l.level
	if l.verbose && effective < LevelDebug {
		effective = LevelDebug
	}
	if level > effective {
		return
	}

	msg := fmt.Sprintf(format, args...)
	prefix := levelNames[level]

	var c *color.Color
	switch level {
	case LevelError:
		c = color.New(color.FgRed, color.Bold)
	case LevelWarn:
		c = color.New(color.FgYellow)
	case LevelDebug:
		c = color.New(color.FgCyan)
	default:
		c = color.New(color.FgGreen)
	}

	_, _ = c.Fprintf(l.writer, "[%s] ", prefix)
	_, _ = fmt.Fprintln(l.writer, msg)
}

func (l *Logger) Debug(format string, args ...any) { l.log(LevelDebug, format, args...) }
func (l *Logger) Info(format string, args ...any)  { l.log(LevelInfo, format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.log(LevelWarn, format, args...) }
func (l *Logger) Error(format string, args ...any) { l.log(LevelError, format, args...) }
func (l *Logger) Trace(format string, args ...any) { l.log(LevelTrace, format, args...) }

func (l *Logger) PrintTable(header []string, rows [][]string) {
	if l.quiet {
		return
	}

	colWidths := make([]int, len(header))
	for i, h := range header {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	for i, h := range header {
		if i > 0 {
			_, _ = fmt.Fprint(l.writer, " | ")
		}
		_, _ = color.New(color.FgWhite, color.Bold).Fprintf(l.writer, "%-*s", colWidths[i], h)
	}
	_, _ = fmt.Fprintln(l.writer)

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				_, _ = fmt.Fprint(l.writer, " | ")
			}
			_, _ = fmt.Fprintf(l.writer, "%-*s", colWidths[i], cell)
		}
		_, _ = fmt.Fprintln(l.writer)
	}
}

func (l *Logger) PrintJSON(v any) {
	if l.quiet {
		return
	}
	data, err := marshalJSON(v)
	if err != nil {
		l.Error("json marshal: %v", err)
		return
	}
	_, _ = fmt.Fprintln(l.writer, string(data))
}
