package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"postcli/internal/schedule"
)

func cmdFlush() *cobra.Command {
	return &cobra.Command{
		Use:   "flush",
		Short: "Process due posts once (for cron/systemd)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			st, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			poster := newAppPoster(st)
			r := &schedule.Runner{Store: st, Poster: poster}
			if err := r.FlushDue(ctx, nowUTC()); err != nil {
				return err
			}
			fmt.Println("postx: flush complete")
			return nil
		},
	}
}
