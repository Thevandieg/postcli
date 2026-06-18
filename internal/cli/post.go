package cli

import (
	"context"

	"github.com/spf13/cobra"

	"postcli/internal/config"
	"postcli/internal/schedule"
	"postcli/internal/tui/post"
)

func cmdPost() *cobra.Command {
	return &cobra.Command{
		Use:   "post",
		Short: "Compose a post (interactive)",
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
			poster := newAppPoster(st)
			runner := &schedule.Runner{Store: st, Poster: poster}
			return post.Run(st, runner)
		},
	}
}
