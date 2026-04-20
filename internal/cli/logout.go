package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"postcli/internal/config"
)

func cmdLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored OAuth tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			st, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.ClearOAuth(ctx, config.TokenPath()); err != nil {
				return err
			}
			fmt.Println("postx: logged out")
			return nil
		},
	}
}
