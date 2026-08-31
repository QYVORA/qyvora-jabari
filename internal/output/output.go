package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	yaml "go.yaml.in/yaml/v3"
)

type Format string

const (
	// FormatTerminal is the human-readable table/text presentation.
	FormatTerminal Format = "terminal"
	// FormatJSON is machine-readable JSON.
	FormatJSON Format = "json"
	// FormatYAML is machine-readable YAML.
	FormatYAML Format = "yaml"
)

// Legacy format names accepted as aliases for FormatTerminal so existing
// scripts and configs keep working.
const (
	FormatTable Format = "table"
	FormatText  Format = "text"
)

// ParseFormat resolves a user-supplied output format name. The legacy names
// "table" and "text" are normalized to "terminal". Unknown formats return a
// useful error rather than being silently accepted.
func ParseFormat(s string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(s))) {
	case FormatTerminal, FormatTable, FormatText:
		return FormatTerminal, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatYAML:
		return FormatYAML, nil
	}
	return "", fmt.Errorf("invalid output format %q: valid values are terminal, json, yaml", s)
}

type Printer struct {
	writer io.Writer
	format Format
	color  bool
}

func New() *Printer {
	return &Printer{
		writer: os.Stdout,
		format: FormatTerminal,
		color:  true,
	}
}

func (p *Printer) SetWriter(w io.Writer) { p.writer = w }
func (p *Printer) SetFormat(f Format)    { p.format = f }
func (p *Printer) SetColor(c bool)       { p.color = c }
func (p *Printer) Format() Format        { return p.format }

func (p *Printer) Print(v any) {
	switch p.format {
	case FormatJSON:
		p.printJSON(v)
	case FormatYAML:
		p.printYAML(v)
	default:
		_, _ = fmt.Fprintln(p.writer, v)
	}
}

func (p *Printer) PrintTable(header []string, rows [][]string) {
	if p.format == FormatJSON {
		entries := make([]map[string]string, len(rows))
		for i, row := range rows {
			entry := make(map[string]string)
			for j, h := range header {
				if j < len(row) {
					entry[h] = row[j]
				}
			}
			entries[i] = entry
		}
		p.printJSON(entries)
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

	headerColor := color.New(color.FgWhite, color.Bold)
	altRowColor := color.New(color.FgBlack, color.BgWhite)

	for i, h := range header {
		if i > 0 {
			_, _ = fmt.Fprint(p.writer, "  ")
		}
		_, _ = headerColor.Fprintf(p.writer, "%-*s", colWidths[i], h)
	}
	_, _ = fmt.Fprintln(p.writer)

	for idx, row := range rows {
		for i, cell := range row {
			if i > 0 {
				_, _ = fmt.Fprint(p.writer, "  ")
			}
			if p.color && idx%2 == 1 {
				_, _ = altRowColor.Fprintf(p.writer, "%-*s", colWidths[i], cell)
			} else {
				_, _ = fmt.Fprintf(p.writer, "%-*s", colWidths[i], cell)
			}
		}
		_, _ = fmt.Fprintln(p.writer)
	}
}

func (p *Printer) printJSON(v any) {
	enc := json.NewEncoder(p.writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "json error: %v\n", err)
	}
}

func (p *Printer) printYAML(v any) {
	out, err := yaml.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "yaml error: %v\n", err)
		return
	}
	_, _ = p.writer.Write(out)
}
