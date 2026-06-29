package schedule

import (
	"context"
	"fmt"
	"strings"

	"postcli/internal/store"
	"postcli/internal/substackapi"
	"postcli/internal/xapi"
	"postcli/internal/xquikapi"
)

// AppPoster routes posts to the correct channel-specific client.
type AppPoster struct {
	XBackend string
	X        *xapi.Client
	Xquik    *xquikapi.Client
	Substack *substackapi.Client
}

func (p *AppPoster) useXquik() bool {
	return strings.EqualFold(strings.TrimSpace(p.XBackend), "xquik")
}

// PostText publishes text content to the requested channel.
func (p *AppPoster) PostText(ctx context.Context, ch store.Channel, text string) (string, error) {
	switch ch {
	case store.ChannelX:
		if p.useXquik() {
			if p.Xquik == nil {
				return "", fmt.Errorf("Xquik client not configured")
			}
			return p.Xquik.PostText(ctx, text)
		}
		if p.X == nil {
			return "", fmt.Errorf("X client not configured")
		}
		return p.X.PostText(ctx, text)
	case store.ChannelSubstack:
		if p.Substack == nil {
			return "", fmt.Errorf("Substack client not configured")
		}
		return p.Substack.PostText(ctx, text)
	default:
		return "", fmt.Errorf("unsupported channel %q", ch)
	}
}

// PostTextWithMedia publishes text and media content to the requested channel.
func (p *AppPoster) PostTextWithMedia(ctx context.Context, ch store.Channel, text, mediaPath string) (string, error) {
	switch ch {
	case store.ChannelX:
		if p.useXquik() {
			if p.Xquik == nil {
				return "", fmt.Errorf("Xquik client not configured")
			}
			return p.Xquik.PostTextWithMedia(ctx, text, mediaPath)
		}
		if p.X == nil {
			return "", fmt.Errorf("X client not configured")
		}
		return p.X.PostTextWithMedia(ctx, text, mediaPath)
	case store.ChannelSubstack:
		if p.Substack == nil {
			return "", fmt.Errorf("Substack client not configured")
		}
		return p.Substack.PostTextWithMedia(ctx, text, mediaPath)
	default:
		return "", fmt.Errorf("unsupported channel %q", ch)
	}
}
