package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion script for the specified shell.

To load completions:

Bash:
  $ source <(qyvora-jabari completion bash)
  # To load completions for each session:
  $ qyvora-jabari completion bash > /etc/bash_completion.d/qyvora-jabari

Zsh:
  $ source <(qyvora-jabari completion zsh)
  # To load completions for each session:
  $ qyvora-jabari completion zsh > "${fpath[1]}/_qyvora-jabari"

Fish:
  $ qyvora-jabari completion fish | source
  # To load completions for each session:
  $ qyvora-jabari completion fish > ~/.config/fish/completions/qyvora-jabari.fish

PowerShell:
  PS> qyvora-jabari completion powershell | Out-String | Invoke-Expression
  PS> qyvora-jabari completion powershell > qyvora-jabari.ps1
`,
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
