package models

import "time"

// Request is a transport-agnostic command to execute against a target.
// Both the USB (ADB) and network transports accept the same shape so the
// assessment engine never needs to know how a command is delivered.
type Request struct {
	// Command is the logical operation name (for example "shell", "getprop",
	// "packages"). Transports map this onto their own primitives.
	Command string
	// Args carries command arguments as delivered to the underlying shell.
	Args []string
	// Stdin is optional input piped into the command.
	Stdin []byte
	// Timeout bounds the entire request; zero means the transport default.
	Timeout time.Duration
}

// Response is the normalized result of a Request. ExitCode, Stdout, and
// Stderr are captured even when a command "fails" so the caller can decide
// how to interpret partial output.
type Response struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	// Duration records how long the command took to complete.
	Duration time.Duration
}

// OK reports whether the transport reported a successful exit.
func (r Response) OK() bool {
	return r.ExitCode == 0
}

// String returns the trimmed stdout as a string, used for the common case of
// parsing single-line responses.
func (r Response) String() string {
	return string(r.Stdout)
}
