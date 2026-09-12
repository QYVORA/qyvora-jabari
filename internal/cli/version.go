package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/QYVORA/qyvora-jabari/internal/output"
	"github.com/QYVORA/qyvora-jabari/internal/version"
)

// newVersionCmd builds the "jabari version" command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Display the version, build commit, build date, build user, and public QYVORA contact details.",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			info := version.GetInfo()

			if printer.Format() != output.FormatTerminal {
				printer.Print(info)
				return nil
			}

			fmt.Printf("jabari %s\n", info.Version)
			fmt.Printf("  framework:  %s\n", info.Framework)
			fmt.Printf("  commit:     %s\n", info.Commit)
			fmt.Printf("  built:      %s\n", info.Date)
			fmt.Printf("  by:         %s\n", info.BuildUser)
			fmt.Printf("  go:         %s %s/%s\n", info.GoVersion, info.OS, info.Arch)
			fmt.Printf("  website:    %s\n", info.Website)
			fmt.Printf("  support:    %s\n", info.Support)
			fmt.Printf("  built in:   %s\n", info.BuiltIn)
			return nil
		},
	}
}
