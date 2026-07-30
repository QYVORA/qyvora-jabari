package utilities

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
)

var reNonAlpha = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func SanitizeName(s string) string {
	return reNonAlpha.ReplaceAllString(s, "-")
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func Indent(text, indent string) string {
	if text == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func Pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func WaitForSignal() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	return <-sigCh
}
