package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// openEventsWriter resolves the --events destination spec into a writer:
//
//	""  and the disable words below   disabled (no event stream)
//	"stdout"  JSONL to stdout (machine output; do not mix with human reports)
//	"stderr"  JSONL to stderr (the default choice for interactive use)
//	anything else is a file path, created/truncated with 0600
//
// The disable words are matched case-insensitively. Without them,
// "--events off" fell through to the file branch and created a file
// literally named "off" in the working directory, so the obvious way to
// turn the stream off created a file instead.
//
// A file destination is truncated rather than appended, so one file holds
// exactly one run's events and has a run boundary at byte zero.
//
// The returned close function must be called when the stream is done; it is
// nil when the stream is disabled or a fixed console.
func openEventsWriter(spec string) (io.Writer, func() error, error) {
	switch strings.ToLower(spec) {
	case "", "off", "none", "disable", "disabled":
		return nil, nil, nil
	case "stdout":
		return os.Stdout, nil, nil
	case "stderr":
		return os.Stderr, nil, nil
	}
	f, err := os.OpenFile(spec, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("events file: %w", err)
	}
	return f, f.Close, nil
}
