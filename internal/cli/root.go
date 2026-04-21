package cli

import (
	"github.com/spf13/cobra"
)

// Execute runs the postx CLI.
func Execute() error {
	root := &cobra.Command{
		Use:   "postx",
		Short: "Schedule and post to X from the terminal",
		Long:  "postx is a Bubble Tea TUI and headless scheduler for X (Twitter) API v2.",
	}
	root.AddCommand(cmdChannels(), cmdLogout(), cmdPost(), cmdStatus(), cmdFlush(), cmdDaemon(), cmdCancel(), cmdTheme())
	root.SilenceUsage = true
	return root.Execute()
}
