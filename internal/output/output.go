package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatText  Format = "text"
	FormatYAML  Format = "yaml"
)

var validFormats = map[Format]bool{
	FormatTable: true,
	FormatJSON:  true,
	FormatText:  true,
	FormatYAML:  true,
}

func ParseFormat(s string) (Format, error) {
	f := Format(strings.ToLower(s))
	if !validFormats[f] {
		return "", fmt.Errorf("invalid output format %q: valid values are table, json, text, yaml", s)
	}
	return f, nil
}

type Printer struct {
	writer io.Writer
	format Format
	color  bool
}

func New() *Printer {
	return &Printer{
		writer: os.Stdout,
		format: FormatTable,
		color:  true,
	}
}

func (p *Printer) SetWriter(w io.Writer)   { p.writer = w }
func (p *Printer) SetFormat(f Format)       { p.format = f }
func (p *Printer) SetColor(c bool)          { p.color = c }
func (p *Printer) Format() Format           { return p.format }

func (p *Printer) Print(v any) {
	switch p.format {
	case FormatJSON:
		p.printJSON(v)
	case FormatYAML:
		p.printYAML(v)
	default:
		fmt.Fprintln(p.writer, v)
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
			fmt.Fprint(p.writer, "  ")
		}
		headerColor.Fprintf(p.writer, "%-*s", colWidths[i], h)
	}
	fmt.Fprintln(p.writer)

	totalWidth := 0
	for i, w := range colWidths {
		if i > 0 {
			totalWidth += 2
		}
		totalWidth += w
	}
	fmt.Fprintln(p.writer, strings.Repeat("─", totalWidth))

	for idx, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(p.writer, "  ")
			}
			if p.color && idx%2 == 1 {
				altRowColor.Fprintf(p.writer, "%-*s", colWidths[i], cell)
			} else {
				fmt.Fprintf(p.writer, "%-*s", colWidths[i], cell)
			}
		}
		fmt.Fprintln(p.writer)
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
	fmt.Fprintln(p.writer, v)
}
