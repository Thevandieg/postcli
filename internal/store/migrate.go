package store

import (
	"context"
	"database/sql"
)

const migrateSQL = `
CREATE TABLE IF NOT EXISTS posts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	kind TEXT NOT NULL,
	channel TEXT NOT NULL DEFAULT 'x',
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
	if _, err := db.ExecContext(ctx, migrateSQL); err != nil {
		return err
	}
	return migratePostsAddChannel(ctx, db)
}

func migratePostsAddChannel(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(posts)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid     int
			name    string
			typ     string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "channel" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `ALTER TABLE posts ADD COLUMN channel TEXT NOT NULL DEFAULT 'x'`)
	return err
}

