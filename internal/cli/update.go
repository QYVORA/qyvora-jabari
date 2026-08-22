// update.go implements `jabari updates`: check the running version against
// jabari's official QYVORA GitHub releases and install a newer release after
// cryptographic verification. See internal/selfupdate for the shared flow.
package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-jabari/internal/output"
	"github.com/QYVORA/qyvora-jabari/internal/selfupdate"
	"github.com/QYVORA/qyvora-jabari/internal/version"
)

// newUpdatesCmd builds the "jabari updates" command. "update" is accepted as
// an alias; "updates" remains the canonical QYVORA verb.
func newUpdatesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "updates",
		Aliases: []string{"update"},
		Short:   "Update jabari from official QYVORA GitHub releases",
		Long: `Check for a newer jabari release and install it.

The installed version is compared against the latest official QYVORA
GitHub release for this platform. If an update exists, it is downloaded,
verified against the release SHA-256 manifest, and swapped in atomically;
the previous binary is never touched unless every step succeeds.

No Go toolchain, Git, or source checkout is required.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := selfupdate.Options{Out: cmd.OutOrStdout()}
			jsonMode := printer.Format() != output.FormatTerminal || quiet
			if jsonMode {
				opts.Quiet = true
			}

			res, err := selfupdate.Run(cmd.Context(), releaseConfig(), opts)
			if err != nil {
				if jsonMode {
					printer.Print(map[string]string{
						"framework": "jabari",
						"command":   "updates",
						"status":    "failed",
						"installed": res.Current,
						"latest":    res.Latest,
						"error":     err.Error(),
					})
				}
				return wrapUpdateError(err)
			}

			if jsonMode {
				status := "updated"
				switch res.Status {
				case selfupdate.StatusCurrent:
					status = "current"
				case selfupdate.StatusNewerInstalled:
					status = "newer_installed"
				}
				payload := map[string]string{
					"framework": "jabari",
					"command":   "updates",
					"status":    status,
					"installed": res.Current,
					"latest":    res.Latest,
				}
				if res.Status == selfupdate.StatusUpdated {
					payload["path"] = res.Path
				}
				printer.Print(payload)
			}
			return nil
		},
	}
}

// releaseConfig pins the updater to jabari's official release source: the
// QYVORA/qyvora-jabari GitHub repository and nothing else.
func releaseConfig() selfupdate.Config {
	return selfupdate.Config{
		Owner:    "QYVORA",
		Repo:     "qyvora-jabari",
		ToolName: "jabari",
		CurrentVersion: func() string {
			return version.Version
		},
		ArtifactName: func(goos, goarch string) string {
			name := fmt.Sprintf("jabari-%s-%s", goos, goarch)
			if goos == "windows" {
				name += ".exe"
			}
			return name
		},
		ChecksumAsset: func(string) string { return "SHA256SUMS" },
	}
}

// wrapUpdateError keeps failures clean for the terminal while expanding
// permission denials into actionable multi-line guidance.
func wrapUpdateError(err error) error {
	var ue *selfupdate.UpdateError
	if !errors.As(err, &ue) {
		return err
	}
	if ue.Kind == selfupdate.KindPermission && ue.Path() != "" {
		return fmt.Errorf("%s\n\n%s", ue.Error(), selfupdate.PermissionHint("jabari", ue.Path()))
	}
	return ue
}
