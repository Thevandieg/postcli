package channels

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"postcli/internal/store"
)

// Status summarizes configuration state for a channel.
type Status struct {
	Entry      Entry
	Configured bool
	Detail     string
}

type Config struct {
	XClientID           string
	XClientSecret       string
	SubstackCookie      string
	SubstackPublication string
}

// Statuses returns user-facing setup status for all channels.
func Statuses(ctx context.Context, st *store.Store, cfg Config) []Status {
	out := make([]Status, 0, len(Catalog()))
	for _, e := range Catalog() {
		s := Status{Entry: e}
		switch e.ID {
		case store.ChannelX:
			idOK := strings.TrimSpace(cfg.XClientID) != ""
			secOK := strings.TrimSpace(cfg.XClientSecret) != ""
			tokenOK := false
			if st != nil {
				_, err := st.LoadOAuth(ctx)
				tokenOK = err == nil
				if err != nil && !errors.Is(err, sql.ErrNoRows) {
					s.Detail = "token check failed"
				}
			}
			s.Configured = idOK && secOK && tokenOK
			if s.Detail == "" {
				switch {
				case !idOK || !secOK:
					s.Detail = "missing credentials"
				case !tokenOK:
					s.Detail = "login required"
				default:
					s.Detail = "configured"
				}
			}
		case store.ChannelSubstack:
			cookieOK := strings.TrimSpace(cfg.SubstackCookie) != ""
			pubOK := strings.TrimSpace(cfg.SubstackPublication) != ""
			s.Configured = cookieOK && pubOK
			if !cookieOK || !pubOK {
				var missing []string
				if !cookieOK {
					missing = append(missing, "cookie")
				}
				if !pubOK {
					missing = append(missing, "publication")
				}
				s.Detail = "missing " + strings.Join(missing, " and ")
			} else {
				s.Detail = "configured"
			}
		default:
			s.Configured = false
			if e.Subtitle != "" {
				s.Detail = e.Subtitle
			} else {
				s.Detail = "coming soon"
			}
		}
		out = append(out, s)
	}
	return out
}
