package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"time"
)

// OAuthToken is persisted for refresh; file backup mirrors DB for portability.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s *Store) SaveOAuth(ctx context.Context, t OAuthToken, tokenPath string) error {
	exp := ""
	if !t.ExpiresAt.IsZero() {
		exp = t.ExpiresAt.UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO oauth_tokens (id, access_token, refresh_token, expires_at, token_type)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			expires_at = excluded.expires_at,
			token_type = excluded.token_type
	`, t.AccessToken, t.RefreshToken, exp, t.TokenType)
	if err != nil {
		return err
	}
	if tokenPath != "" {
		b, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(tokenPath, b, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) LoadOAuth(ctx context.Context) (OAuthToken, error) {
	var t OAuthToken
	var access, refresh, typ, exp sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT access_token, refresh_token, expires_at, token_type FROM oauth_tokens WHERE id = 1
	`).Scan(&access, &refresh, &exp, &typ)
	if err == sql.ErrNoRows {
		return OAuthToken{}, sql.ErrNoRows
	}
	if err != nil {
		return OAuthToken{}, err
	}
	t.AccessToken = access.String
	t.RefreshToken = refresh.String
	t.TokenType = typ.String
	if exp.Valid && exp.String != "" {
		t.ExpiresAt, err = time.Parse(time.RFC3339, exp.String)
		if err != nil {
			return OAuthToken{}, err
		}
	}
	return t, nil
}

func (s *Store) ClearOAuth(ctx context.Context, tokenPath string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oauth_tokens WHERE id = 1`)
	if err != nil {
		return err
	}
	if tokenPath != "" {
		_ = os.Remove(tokenPath)
	}
	return nil
}
