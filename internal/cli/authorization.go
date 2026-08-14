package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	errs "github.com/QYVORA/qyvora-jabari/internal/errors"
	"github.com/QYVORA/qyvora-jabari/pkg/models"
	"github.com/spf13/cobra"
)

// authorizationFlags are shared by every command that establishes a target.
var authorizationFlags struct {
	// authorized confirms authorization non-interactively. Intended for
	// automation where a human has already approved the scope.
	authorized bool
}

// authorize gates a target behind explicit authorization. The gate is
// satisfied (in order) by the --authorized/-y flag, the QYVORA_AUTHORIZED
// environment variable, an interactive confirmation when stdin is a terminal,
// or a hard failure in non-interactive contexts.
//
// It returns a copy of the target with the authorization recorded.
func authorize(cmd *cobra.Command, t *models.Target) (*models.Target, error) {
	switch {
	case authorizationFlags.authorized || cfg.GetBool("authorized"):
		t.Auth = granted(t)
		return t, nil
	case strings.EqualFold(os.Getenv("QYVORA_AUTHORIZED"), "true"):
		t.Auth = granted(t)
		return t, nil
	}

	if isTTY(os.Stdin) {
		fmt.Fprintf(os.Stderr, "\nAuthorized Android security assessment\n")
		fmt.Fprintf(os.Stderr, "  Target: %s (%s)\n", t.DisplayName(), t.Type)
		fmt.Fprintf(os.Stderr, "  Scope:  authorized assessment only, scoped to this target\n")
		fmt.Fprintf(os.Stderr, "Confirm authorization? [y/N] ")
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err == nil && strings.EqualFold(strings.TrimSpace(answer), "y") {
			t.Auth = granted(t)
			return t, nil
		}
		return nil, errs.NewExitError(3, "authorization declined; assessment aborted")
	}

	return nil, errs.NewExitError(3,
		"target authorization required; re-run with --authorized to confirm scope non-interactively")
}

func granted(t *models.Target) models.Authorization {
	return models.Authorization{
		Granted:   true,
		Scope:     "authorized Android security assessment of " + t.DisplayName(),
		Method:    "cli",
		GrantedBy: user(),
	}
}

func user() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return "unknown"
}

// isTTY reports whether w is an interactive terminal. Non-TTY stdin means an
// interactive prompt would hang, so authorization must come from a flag or
// environment.
func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
