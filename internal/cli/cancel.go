package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func cmdCancel() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel ID",
		Short: "Cancel a pending scheduled post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id: %w", err)
			}
			st, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer st.Close()
			if err := st.MarkCancelled(ctx, id); err != nil {
				return err
			}
			fmt.Printf("postx: cancelled #%d\n", id)
			return nil
		},
	}
}
