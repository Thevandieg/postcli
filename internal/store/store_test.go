package store

import (
	"context"
	"testing"
	"time"
)

func TestInsertAndListMonth(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir()+"/q.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewStore(db)

	payload := PostPayload{Text: "hello"}
	at := time.Date(2026, 4, 21, 15, 30, 0, 0, time.UTC)
	id, err := s.InsertPost(ctx, KindText, payload, at, StatusPending, "k1")
	if err != nil {
		t.Fatal(err)
	}
	if id < 1 {
		t.Fatalf("id %d", id)
	}

	month := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	list, err := s.ListPostsInMonth(ctx, month)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d posts", len(list))
	}
	if list[0].Payload.Text != "hello" || list[0].Status != StatusPending {
		t.Fatalf("%+v", list[0])
	}
}

func TestDuePending(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir()+"/q.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewStore(db)

	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)
	_, err = s.InsertPost(ctx, KindText, PostPayload{Text: "a"}, past, StatusPending, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.InsertPost(ctx, KindText, PostPayload{Text: "b"}, future, StatusPending, "")
	if err != nil {
		t.Fatal(err)
	}

	due, err := s.ListDuePending(ctx, time.Now().UTC(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].Payload.Text != "a" {
		t.Fatalf("%+v", due)
	}
}

func TestCancel(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, t.TempDir()+"/q.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewStore(db)
	id, err := s.InsertPost(ctx, KindText, PostPayload{Text: "x"}, time.Now().UTC(), StatusPending, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkCancelled(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkCancelled(ctx, id); err == nil {
		t.Fatal("expected error cancelling non-pending")
	}
}
