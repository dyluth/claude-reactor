package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"claude-reactor/pkg"
)

// NewCompletionCmd creates the completion command with installation instructions
func NewCompletionCmd(app *pkg.AppContainer) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion scripts for your shell",
		Long: `Generate completion scripts for claude-reactor commands.

To load completions:

Bash:
  $ source <(claude-reactor completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ claude-reactor completion bash > /etc/bash_completion.d/claude-reactor
  # macOS:
  $ claude-reactor completion bash > /usr/local/etc/bash_completion.d/claude-reactor

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ claude-reactor completion zsh > "${fpath[1]}/_claude-reactor"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ claude-reactor completion fish | source

  # To load completions for each session, execute once:
  $ claude-reactor completion fish > ~/.config/fish/completions/claude-reactor.fish

PowerShell:
  PS> claude-reactor completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> claude-reactor completion powershell > claude-reactor.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.ExactValidArgs(1),
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
				return fmt.Errorf("unsupported shell type %q", args[0])
			}
		},
	}

	return completionCmd
}