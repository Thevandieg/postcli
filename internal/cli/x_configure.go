package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"postcli/internal/config"
	"postcli/internal/tui/xconfigureui"
)

func runXConfigureWizard(ctx context.Context, timeout time.Duration) error {
	return xconfigureui.Run(xconfigureui.Deps{
		Ctx:          ctx,
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		RedirectURI:  RedirectURI,
		LoadEnvMap:   config.LoadEnvMap,
		PersistEnv:   persistEnvAndShell,
		ApplyEnv:     applyEnv,
		OAuthLogin: func(ctx context.Context, skipBrowser bool) error {
			return xOAuthLogin(ctx, timeout, skipBrowser)
		},
		LoginStatus: func(ctx context.Context) (bool, string, error) {
			st, err := openStore(ctx)
			if err != nil {
				return false, "", err
			}
			defer st.Close()
			_, err = st.LoadOAuth(ctx)
			if err == nil {
				return true, "token on disk", nil
			}
			if errors.Is(err, sql.ErrNoRows) {
				return false, "not logged in yet", nil
			}
			return false, err.Error(), err
		},
	})
}

func xOAuthLogin(ctx context.Context, timeout time.Duration, skipBrowser bool) error {
	if err := requireXAppCredentials(); err != nil {
		return err
	}
	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()
	loginCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := doXLogin(loginCtx, st, RedirectURI(), timeout, skipBrowser); err != nil {
		return err
	}
	return nil
}

func persistEnvAndShell(envMap map[string]string) error {
	if err := config.EnsureDir(); err != nil {
		return err
	}
	if err := config.SaveEnvMap(envMap); err != nil {
		return err
	}
	if err := syncShellProfile(envMap); err != nil {
		fmt.Fprintf(os.Stderr, "postx: warning: shell profile update failed: %v\n", err)
	}
	return nil
}

func applyEnv(envMap map[string]string) {
	if v := strings.TrimSpace(envMap["POSTX_CLIENT_ID"]); v != "" {
		_ = os.Setenv("POSTX_CLIENT_ID", v)
	}
	if v := strings.TrimSpace(envMap["POSTX_CLIENT_SECRET"]); v != "" {
		_ = os.Setenv("POSTX_CLIENT_SECRET", v)
	}
	if v := strings.TrimSpace(envMap["POSTX_REDIRECT_URI"]); v != "" {
		_ = os.Setenv("POSTX_REDIRECT_URI", v)
	}
}

func requireXAppCredentials() error {
	if strings.TrimSpace(ClientID()) == "" || strings.TrimSpace(ClientSecret()) == "" {
		return fmt.Errorf("set Client ID and Client Secret first (menu options 1–3)")
	}
	return nil
}
