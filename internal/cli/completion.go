package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// newCompletionCmd builds the "jabari completion" command. It generates
// completion scripts under the binary's current command name so the alias
// "androidsec" produces the same completions.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion script for the specified shell.

To load completions:

Bash:
  $ source <(jabari completion bash)
  $ jabari completion bash > /etc/bash_completion.d/jabari

Zsh:
  $ source <(jabari completion zsh)
  $ jabari completion zsh > "${fpath[1]}/_jabari"

Fish:
  $ jabari completion fish | source

PowerShell:
  PS> jabari completion powershell | Out-String | Invoke-Expression`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}
