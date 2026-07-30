package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

type Level int

const (
	LevelError Level = iota
	LevelWarn
	LevelInfo
	LevelDebug
)

var levelNames = map[Level]string{
	LevelError: "ERROR",
	LevelWarn:  "WARN",
	LevelInfo:  "INFO",
	LevelDebug: "DEBUG",
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

func (l *Logger) SetVerbose(v bool)    { l.verbose = v }
func (l *Logger) SetQuiet(q bool)       { l.quiet = q }
func (l *Logger) SetLevel(level Level)  { l.level = level }
func (l *Logger) SetWriter(w io.Writer) { l.writer = w }

func (l *Logger) log(level Level, format string, args ...any) {
	if l.quiet && level < LevelError {
		return
	}
	if level > l.level && !(level == LevelDebug && l.verbose) {
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

	c.Fprintf(l.writer, "[%s] ", prefix)
	fmt.Fprintln(l.writer, msg)
}

func (l *Logger) Debug(format string, args ...any) { l.log(LevelDebug, format, args...) }
func (l *Logger) Info(format string, args ...any)   { l.log(LevelInfo, format, args...) }
func (l *Logger) Warn(format string, args ...any)   { l.log(LevelWarn, format, args...) }
func (l *Logger) Error(format string, args ...any)   { l.log(LevelError, format, args...) }

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

	sep := func() {
		for i, w := range colWidths {
			if i > 0 {
				fmt.Fprint(l.writer, "-+-")
			}
			fmt.Fprint(l.writer, strings.Repeat("-", w))
		}
		fmt.Fprintln(l.writer)
	}

	for i, h := range header {
		if i > 0 {
			fmt.Fprint(l.writer, " | ")
		}
		color.New(color.FgWhite, color.Bold).Fprintf(l.writer, "%-*s", colWidths[i], h)
	}
	fmt.Fprintln(l.writer)
	sep()

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(l.writer, " | ")
			}
			fmt.Fprintf(l.writer, "%-*s", colWidths[i], cell)
		}
		fmt.Fprintln(l.writer)
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
	fmt.Fprintln(l.writer, string(data))
}
