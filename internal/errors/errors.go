package errors

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

type ExitError struct {
	Code    int
	Message string
	Cause   error
}

func (e *ExitError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *ExitError) Unwrap() error {
	return e.Cause
}

func NewExitError(code int, msg string) *ExitError {
	return &ExitError{Code: code, Message: msg}
}

func WrapExitError(code int, msg string, cause error) *ExitError {
	return &ExitError{Code: code, Message: msg, Cause: cause}
}

func Fatalf(code int, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	_, _ = color.New(color.FgRed, color.Bold).Fprintf(os.Stderr, "Error: ")
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}

func Fatalln(code int, msg string) {
	_, _ = color.New(color.FgRed, color.Bold).Fprintf(os.Stderr, "Error: ")
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "not found")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
