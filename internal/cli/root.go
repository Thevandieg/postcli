package cli

import (
	"github.com/spf13/cobra"
)

// Execute runs the postx CLI.
func Execute() error {
	root := &cobra.Command{
		Use:   "postx",
		Short: "Minimal CLI to compose, schedule, and publish social posts",
		Long:  "postx is a minimal CLI to compose, schedule, and publish social posts.",
	}
	root.AddCommand(cmdChannels(), cmdLogout(), cmdPost(), cmdStatus(), cmdFlush(), cmdDaemon(), cmdCancel(), cmdTheme())
	root.SilenceUsage = true
	return root.Execute()
}
