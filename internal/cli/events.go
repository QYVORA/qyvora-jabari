package cli

import (
	"fmt"
	"io"
	"os"
)

// openEventsWriter resolves the --events destination spec into a writer:
//
//	""        disabled (no event stream)
//	"stdout"  JSONL to stdout (machine output; do not mix with human reports)
//	"stderr"  JSONL to stderr (the default choice for interactive use)
//	anything else is a file path, created/truncated with 0600
//
// The returned close function must be called when the stream is done; it is
// nil when the stream is disabled or a fixed console.
func openEventsWriter(spec string) (io.Writer, func() error, error) {
	switch spec {
	case "":
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
