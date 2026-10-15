// Package cmd wires gh-clip's cobra command tree.
package cmd

import (
	"github.com/spf13/cobra"
)

// Version is injected at build time (see main.go / .github/workflows/release.yml).
var Version = "dev"

// NewRootCmd builds the `gh clip` root command and all its subcommands.
//
// The root command itself is the clipboard: piped input is pushed, and with
// no input the interactive picker opens.
func NewRootCmd() *cobra.Command {
	var share bool

	root := &cobra.Command{
		Use:   "clip",
		Short: "A cross-device clipboard backed by GitHub",
		Long: "gh-clip is a serverless clipboard for the GitHub CLI.\n\n" +
			"  echo \"secret\" | gh clip          save to your private storage repo\n" +
			"  echo \"link\"   | gh clip --share  share to your phone via QR code\n" +
			"  gh clip                           pick a saved clip and copy it",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if stdinIsPipe() {
				if share {
					return runShare(cmd.InOrStdin())
				}
				return runPush(cmd.InOrStdin())
			}
			if share {
				return errShareNeedsInput
			}
			return runPull()
		},
	}
	root.Flags().BoolVar(&share, "share", false, "share piped input to your phone via a secret Gist and QR code")

	root.AddCommand(newInitCmd())
	root.AddCommand(newSweepCmd())
	root.AddCommand(newConfigCmd())
	return root
}
