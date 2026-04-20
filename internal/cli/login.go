package cli

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"

	"postcli/internal/config"
	"postcli/internal/store"
	"postcli/internal/xapi"
)

func cmdLogin() *cobra.Command {
	var redirect string
	var timeout time.Duration
	c := &cobra.Command{
		Use:   "login",
		Short: "OAuth 2.0 login (opens browser)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			if err := config.EnsureDir(); err != nil {
				return err
			}
			st, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer st.Close()

			redir := redirect
			if redir == "" {
				redir = RedirectURI()
			}
			u, err := url.Parse(redir)
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

			o := xapi.OAuthConfig{
				ClientID:     ClientID(),
				ClientSecret: ClientSecret(),
				RedirectURI:  redir,
			}
			fmt.Fprintf(os.Stderr, "postx: login timeout is %v (use --timeout to change)\n", timeout)
			tok, err := o.LoginInteractive(ctx, listenAddr)
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
		},
	}
	c.Flags().StringVar(&redirect, "redirect", "", "OAuth redirect URI (default POSTX_REDIRECT_URI or http://127.0.0.1:8080/callback)")
	c.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "max time to wait for browser redirect to callback URL")
	return c
}
