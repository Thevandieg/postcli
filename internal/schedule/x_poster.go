package schedule

import (
	"context"
	"fmt"

	"postcli/internal/store"
	"postcli/internal/xapi"
)

// XChannelPoster posts only to X (Twitter); other channels return a clear error.
type XChannelPoster struct {
	X *xapi.Client
}

func (p *XChannelPoster) PostText(ctx context.Context, ch store.Channel, text string) (string, error) {
	if p == nil || p.X == nil {
		return "", fmt.Errorf("X client not configured")
	}
	if ch != store.ChannelX {
		return "", fmt.Errorf("unsupported channel %q (integration not available yet)", ch)
	}
	return p.X.PostText(ctx, text)
}

func (p *XChannelPoster) PostTextWithMedia(ctx context.Context, ch store.Channel, text, mediaPath string) (string, error) {
	if p == nil || p.X == nil {
		return "", fmt.Errorf("X client not configured")
	}
	if ch != store.ChannelX {
		return "", fmt.Errorf("unsupported channel %q (integration not available yet)", ch)
	}
	return p.X.PostTextWithMedia(ctx, text, mediaPath)
}
