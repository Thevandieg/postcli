package schedule

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"postcli/internal/store"
)

// Poster creates tweets (implemented by xapi.Client).
type Poster interface {
	PostText(ctx context.Context, text string) (tweetID string, err error)
	PostTextWithMedia(ctx context.Context, text string, mediaPath string) (tweetID string, err error)
}

// Runner processes due posts using a Poster.
type Runner struct {
	Store  *store.Store
	Poster Poster
}

// FlushDue picks pending posts with scheduled_at <= now and posts them.
func (r *Runner) FlushDue(ctx context.Context, now time.Time) error {
	posts, err := r.Store.ListDuePending(ctx, now, 50)
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range posts {
		if err := r.processOne(ctx, p); err != nil {
			fmt.Fprintf(os.Stderr, "postx: post %d: %v\n", p.ID, err)
			errs = append(errs, fmt.Errorf("post %d: %w", p.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (r *Runner) processOne(ctx context.Context, p store.Post) error {
	if err := r.Store.MarkPosting(ctx, p.ID); err != nil {
		return err
	}
	var tweetID string
	var err error
	switch p.Kind {
	case store.KindText:
		tweetID, err = r.Poster.PostText(ctx, p.Payload.Text)
	case store.KindTextWithMedia:
		tweetID, err = r.Poster.PostTextWithMedia(ctx, p.Payload.Text, p.Payload.MediaPath)
	default:
		err = fmt.Errorf("unknown kind %q", p.Kind)
	}
	if err != nil {
		_ = r.Store.MarkFailed(ctx, p.ID, err.Error())
		return err
	}
	return r.Store.MarkPosted(ctx, p.ID, tweetID)
}
