package cli

import (
	"context"
	"fmt"

	"postcli/internal/xapi"
)

func ensurePostingReady(ctx context.Context, client *xapi.Client) error {
	if err := client.CheckReady(ctx); err != nil {
		return fmt.Errorf("cannot post yet: %s", xapi.UserMessage(err))
	}
	return nil
}
