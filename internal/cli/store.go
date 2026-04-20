package cli

import (
	"context"

	"postcli/internal/config"
	"postcli/internal/store"
)

func openStore(ctx context.Context) (*store.Store, error) {
	if err := config.EnsureDir(); err != nil {
		return nil, err
	}
	db, err := store.Open(ctx, config.DBPath())
	if err != nil {
		return nil, err
	}
	return store.NewStore(db), nil
}
