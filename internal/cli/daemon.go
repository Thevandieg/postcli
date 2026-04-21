package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"postcli/internal/config"
	"postcli/internal/schedule"
	"postcli/internal/xapi"
)

func cmdDaemon() *cobra.Command {
	var interval time.Duration
	c := &cobra.Command{
		Use:   "daemon",
		Short: "Poll for due posts and post them",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			st, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			client := &xapi.Client{
				OAuth: xapi.OAuthConfig{
					ClientID:     ClientID(),
					ClientSecret: ClientSecret(),
					RedirectURI:  RedirectURI(),
				},
				TokenStore: st,
				TokenPath:  config.TokenPath(),
				DryRun:     DryRun(),
			}
			poster := &schedule.XChannelPoster{X: client}
			r := &schedule.Runner{Store: st, Poster: poster}
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				if err := r.FlushDue(ctx, nowUTC()); err != nil {
					fmt.Fprintf(os.Stderr, "postx: flush: %v\n", err)
				}
				select {
				case <-ctx.Done():
					return nil
				case <-t.C:
				}
			}
		},
	}
	c.Flags().DurationVar(&interval, "interval", 20*time.Second, "poll interval")
	return c
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
