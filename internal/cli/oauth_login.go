package cli

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"postcli/internal/config"
	"postcli/internal/store"
	"postcli/internal/xapi"
)

func doXLogin(ctx context.Context, st *store.Store, redirect string, timeout time.Duration, skipBrowser bool) error {
	u, err := url.Parse(redirect)
	if err != nil {
		return fmt.Errorf("redirect URI: %w", err)
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	// Bind 0.0.0.0 so WSL2 + Windows browser works: Windows forwards
	// localhost:port into the distro; a 127.0.0.1-only bind often never
	// sees that traffic. OAuth still uses the exact redirect_uri string.
	listenAddr := net.JoinHostPort("0.0.0.0", port)
	fmt.Fprintf(os.Stderr, "postx: login timeout is %v (use --timeout to change)\n", timeout)
	o := makeOAuthConfig(redirect)
	var tok xapi.Token
	if skipBrowser {
		tok, err = o.LoginInteractiveNoBrowser(ctx, listenAddr)
	} else {
		tok, err = o.LoginInteractive(ctx, listenAddr)
	}
	if err != nil {
		return err
	}
	ot := store.OAuthToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		ExpiresAt:    tok.ExpiresAt,
	}
	if err := st.SaveOAuth(ctx, ot, config.TokenPath()); err != nil {
		return err
	}
	fmt.Println("postx: saved credentials to", config.TokenPath())
	return nil
}
