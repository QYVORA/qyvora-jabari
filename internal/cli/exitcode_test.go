package cli

import (
	"testing"
)

// TestUsageErrorsExitTwo verifies the shared QYVORA exit-code contract: flag
// errors and unknown commands are usage errors (exit 2), not runtime errors
// (exit 1). Regression test for the contract drift where both exited 1.
func TestUsageErrorsExitTwo(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"--bogus"}},
		{"unknown command", []string{"frobnicate"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := ExecuteArgs(tc.args)
			if code != 2 {
				t.Fatalf("exit code for %v = %d, want 2", tc.args, code)
			}
		})
	}
}

// TestVersionExitsZero keeps the happy-path exit code pinned.
func TestVersionExitsZero(t *testing.T) {
	if code := ExecuteArgs([]string{"version"}); code != 0 {
		t.Fatalf("version exit code = %d, want 0", code)
	}
}
