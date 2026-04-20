package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type PostKind string

const (
	KindText          PostKind = "text"
	KindTextWithMedia PostKind = "text_with_media"
)

type PostStatus string

const (
	StatusPending   PostStatus = "pending"
	StatusPosting   PostStatus = "posting"
	StatusPosted    PostStatus = "posted"
	StatusFailed    PostStatus = "failed"
	StatusCancelled PostStatus = "cancelled"
)

// PostPayload is stored as JSON in the posts table.
type PostPayload struct {
	Text      string `json:"text"`
	MediaPath string `json:"media_path,omitempty"`
}

type Post struct {
	ID               int64
	Kind             PostKind
	Payload          PostPayload
	ScheduledAt      time.Time
	Status           PostStatus
	IdempotencyKey   string
	LastError        string
	TweetID          string
	CreatedAt        time.Time
}

func (s *Store) InsertPost(ctx context.Context, kind PostKind, payload PostPayload, scheduledAt time.Time, status PostStatus, idempotencyKey string) (int64, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO posts (kind, payload, scheduled_at, status, idempotency_key, last_error, tweet_id, created_at)
		VALUES (?, ?, ?, ?, ?, '', '', ?)
	`, string(kind), string(b), scheduledAt.UTC().Format(time.RFC3339), string(status), nullIfEmpty(idempotencyKey), now.Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (s *Store) ListPostsForDay(ctx context.Context, day time.Time) ([]Post, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, payload, scheduled_at, status, idempotency_key, last_error, tweet_id, created_at
		FROM posts
		WHERE scheduled_at >= ? AND scheduled_at < ? AND status != 'cancelled'
		ORDER BY scheduled_at
	`, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

// ListPostsInMonth returns all non-cancelled posts scheduled in the given calendar month (UTC).
func (s *Store) ListPostsInMonth(ctx context.Context, month time.Time) ([]Post, error) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, payload, scheduled_at, status, idempotency_key, last_error, tweet_id, created_at
		FROM posts
		WHERE scheduled_at >= ? AND scheduled_at < ? AND status != 'cancelled'
		ORDER BY scheduled_at
	`, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func (s *Store) ListPendingInRange(ctx context.Context, from, to time.Time) ([]Post, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, payload, scheduled_at, status, idempotency_key, last_error, tweet_id, created_at
		FROM posts
		WHERE scheduled_at >= ? AND scheduled_at < ? AND status IN ('pending', 'posting', 'failed')
		ORDER BY scheduled_at
	`, from.Format(time.RFC3339), to.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func scanPostRow(row *sql.Row) (Post, error) {
	var (
		p                        Post
		kind, pay, sched, stat   string
		idemNull                 sql.NullString
		lastErr, tweetID, created string
	)
	err := row.Scan(&p.ID, &kind, &pay, &sched, &stat, &idemNull, &lastErr, &tweetID, &created)
	if err != nil {
		return Post{}, err
	}
	return assemblePost(p, kind, pay, sched, stat, idemNull, lastErr, tweetID, created)
}

func assemblePost(p Post, kind, pay, sched, stat string, idemNull sql.NullString, lastErr, tweetID, created string) (Post, error) {
	p.Kind = PostKind(kind)
	if err := json.Unmarshal([]byte(pay), &p.Payload); err != nil {
		return Post{}, err
	}
	var err error
	p.ScheduledAt, err = time.Parse(time.RFC3339, sched)
	if err != nil {
		return Post{}, err
	}
	p.Status = PostStatus(stat)
	if idemNull.Valid {
		p.IdempotencyKey = idemNull.String
	}
	p.LastError = lastErr
	p.TweetID = tweetID
	p.CreatedAt, err = time.Parse(time.RFC3339, created)
	if err != nil {
		return Post{}, err
	}
	return p, nil
}

func scanPosts(rows *sql.Rows) ([]Post, error) {
	var out []Post
	for rows.Next() {
		var (
			p                        Post
			kind, pay, sched, stat   string
			idemNull                 sql.NullString
			lastErr, tweetID, created string
		)
		if err := rows.Scan(&p.ID, &kind, &pay, &sched, &stat, &idemNull, &lastErr, &tweetID, &created); err != nil {
			return nil, err
		}
		p, err := assemblePost(p, kind, pay, sched, stat, idemNull, lastErr, tweetID, created)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPost returns one post by id.
func (s *Store) GetPost(ctx context.Context, id int64) (Post, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, kind, payload, scheduled_at, status, idempotency_key, last_error, tweet_id, created_at
		FROM posts WHERE id = ?
	`, id)
	p, err := scanPostRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Post{}, fmt.Errorf("post %d: not found", id)
		}
		return Post{}, err
	}
	return p, nil
}

func (s *Store) ListDuePending(ctx context.Context, now time.Time, limit int) ([]Post, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, payload, scheduled_at, status, idempotency_key, last_error, tweet_id, created_at
		FROM posts
		WHERE status = 'pending' AND scheduled_at <= ?
		ORDER BY scheduled_at
		LIMIT ?
	`, now.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPosts(rows)
}

func (s *Store) MarkPosting(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE posts SET status = 'posting', last_error = '' WHERE id = ? AND status = 'pending'`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("post not pending or missing")
	}
	return nil
}

func (s *Store) MarkPosted(ctx context.Context, id int64, tweetID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE posts SET status = 'posted', tweet_id = ?, last_error = '' WHERE id = ?`, tweetID, id)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, id int64, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE posts SET status = 'failed', last_error = ? WHERE id = ?`, errMsg, id)
	return err
}

func (s *Store) MarkCancelled(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE posts SET status = 'cancelled' WHERE id = ? AND status = 'pending'`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("post %d not pending or missing", id)
	}
	return nil
}

// Store wraps DB with app queries.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}
