package store

import (
	"context"
	"database/sql"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS posts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	kind TEXT NOT NULL,
	payload TEXT NOT NULL,
	scheduled_at TEXT NOT NULL,
	status TEXT NOT NULL,
	idempotency_key TEXT UNIQUE,
	last_error TEXT,
	tweet_id TEXT,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_posts_scheduled ON posts(status, scheduled_at);

CREATE TABLE IF NOT EXISTS oauth_tokens (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	access_token TEXT,
	refresh_token TEXT,
	expires_at TEXT,
	token_type TEXT
);
`

func Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, migrateSQL)
	return err
}
