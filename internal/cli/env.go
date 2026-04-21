package cli

import (
	"os"
	"strings"

	"postcli/internal/config"
)

func ClientID() string {
	if v := strings.TrimSpace(os.Getenv("POSTX_CLIENT_ID")); v != "" {
		return v
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		return strings.TrimSpace(kv["POSTX_CLIENT_ID"])
	}
	return ""
}

func ClientSecret() string {
	if v := strings.TrimSpace(os.Getenv("POSTX_CLIENT_SECRET")); v != "" {
		return v
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		return strings.TrimSpace(kv["POSTX_CLIENT_SECRET"])
	}
	return ""
}

func RedirectURI() string {
	if s := strings.TrimSpace(os.Getenv("POSTX_REDIRECT_URI")); s != "" {
		return s
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		if s := strings.TrimSpace(kv["POSTX_REDIRECT_URI"]); s != "" {
			return s
		}
	}
	return "http://127.0.0.1:8080/callback"
}

func DryRun() bool {
	v := strings.TrimSpace(os.Getenv("POSTX_DRY_RUN"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}
