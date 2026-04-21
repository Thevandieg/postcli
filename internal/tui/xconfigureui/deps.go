package xconfigureui

import "context"

// Deps supplies environment and side effects from the CLI layer (avoids import cycles).
type Deps struct {
	Ctx context.Context

	ClientID     func() string
	ClientSecret func() string
	RedirectURI  func() string

	LoadEnvMap func() (map[string]string, error)
	PersistEnv func(values map[string]string) error
	ApplyEnv   func(values map[string]string)

	OAuthLogin func(ctx context.Context, skipBrowser bool) error

	// LoginStatus reports whether a token exists and a short detail string.
	LoginStatus func(ctx context.Context) (hasToken bool, detail string, err error)
}
