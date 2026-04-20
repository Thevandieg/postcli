package cli

import (
	"context"

	"github.com/spf13/cobra"

	"postcli/internal/config"
	"postcli/internal/tui/status"
)

func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Calendar view of scheduled posts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if err := config.EnsureDir(); err != nil {
				return err
			}
			st, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			return status.Run(st)
		},
	}
}
