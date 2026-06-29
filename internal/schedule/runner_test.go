package schedule

import (
	"context"
	"errors"
	"testing"
	"time"

	"postcli/internal/store"
	"postcli/internal/xapi"
	"postcli/internal/xquikapi"
)

type fakePoster struct {
	id  string
	err error
}

func (f fakePoster) PostText(ctx context.Context, ch store.Channel, text string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.id, nil
}

func (f fakePoster) PostTextWithMedia(ctx context.Context, ch store.Channel, text string, mediaPath string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.id + "-m", nil
}

func TestRunnerPosted(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(ctx, t.TempDir()+"/q.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st := store.NewStore(db)

	when := time.Now().UTC().Add(-time.Minute)
	id, err := st.InsertPost(ctx, store.ChannelX, store.KindText, store.PostPayload{Text: "hi"}, when, store.StatusPending, "")
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{Store: st, Poster: fakePoster{id: "42"}}
	if err := r.FlushDue(ctx, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	due, err := st.ListDuePending(ctx, time.Now().UTC(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 0 {
		t.Fatalf("expected empty due, got %+v", due)
	}
	day := time.Date(when.Year(), when.Month(), when.Day(), 0, 0, 0, 0, time.UTC)
	posts, err := st.ListPostsForDay(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, p := range posts {
		if p.ID == id && p.Status == store.StatusPosted && p.TweetID == "42" {
			found = true
		}
	}
	if !found {
		t.Fatal("post not marked posted")
	}
}

func TestRunnerFailed(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(ctx, t.TempDir()+"/q.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	st := store.NewStore(db)
	when := time.Now().UTC().Add(-time.Minute)
	id, err := st.InsertPost(ctx, store.ChannelX, store.KindText, store.PostPayload{Text: "x"}, when, store.StatusPending, "")
	if err != nil {
		t.Fatal(err)
	}
	r := &Runner{Store: st, Poster: fakePoster{err: errors.New("boom")}}
	if err := r.FlushDue(ctx, time.Now().UTC()); err == nil {
		t.Fatal("expected FlushDue error")
	}
	day := time.Date(when.Year(), when.Month(), when.Day(), 0, 0, 0, 0, time.UTC)
	posts, err := st.ListPostsForDay(ctx, day)
	if err != nil {
		t.Fatal(err)
	}
	var p *store.Post
	for i := range posts {
		if posts[i].ID == id {
			p = &posts[i]
			break
		}
	}
	if p == nil || p.Status != store.StatusFailed || p.LastError == "" {
		t.Fatalf("%+v", p)
	}
}

func TestAppPosterUsesDefaultXBackend(t *testing.T) {
	p := &AppPoster{
		X: &xapi.Client{DryRun: true},
	}
	id, err := p.PostText(context.Background(), store.ChannelX, "hello")
	if err != nil {
		t.Fatalf("PostText returned error: %v", err)
	}
	if id != "dry-run-id" {
		t.Fatalf("id = %q; want dry-run-id", id)
	}
}

func TestAppPosterUsesXquikBackendWhenSelected(t *testing.T) {
	p := &AppPoster{
		XBackend: "xquik",
		Xquik: &xquikapi.Client{
			Account: "@acct",
			DryRun:  true,
		},
	}
	id, err := p.PostText(context.Background(), store.ChannelX, "hello")
	if err != nil {
		t.Fatalf("PostText returned error: %v", err)
	}
	if id != "dry-run-xquik-id" {
		t.Fatalf("id = %q; want dry-run-xquik-id", id)
	}
}
