// Package command builds the taiichi command tree.
package command

import "github.com/spf13/cobra"

// NewRoot returns the taiichi command with every subcommand attached.
func NewRoot() *cobra.Command {
	return &cobra.Command{
		Use:   "taiichi",
		Short: "Move Jobs through the Pipeline, from Pickup to Done",
		// Fang sets these too, but tests run the tree without Fang.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}
