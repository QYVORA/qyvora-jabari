package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anomalyco/qyvora-jabari/internal/output"
	"github.com/anomalyco/qyvora-jabari/internal/version"
)

// newVersionCmd builds the "jabari version" command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Display the version, build commit, build date, and build user.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.GetInfo()

			if printer.Format() == output.FormatJSON {
				printer.Print(info)
				return nil
			}

			fmt.Printf("jabari %s\n", info.Version)
			fmt.Printf("  Commit:    %s\n", info.Commit)
			fmt.Printf("  Built:     %s\n", info.Date)
			fmt.Printf("  BuildUser: %s\n", info.BuildUser)
			return nil
		},
	}
}
