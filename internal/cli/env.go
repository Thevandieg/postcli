package cli

import (
	"os"
	"strings"
)

func ClientID() string {
	return strings.TrimSpace(os.Getenv("POSTX_CLIENT_ID"))
}

func ClientSecret() string {
	return strings.TrimSpace(os.Getenv("POSTX_CLIENT_SECRET"))
}

func RedirectURI() string {
	s := strings.TrimSpace(os.Getenv("POSTX_REDIRECT_URI"))
	if s != "" {
		return s
	}
	return "http://127.0.0.1:8080/callback"
}

func DryRun() bool {
	v := strings.TrimSpace(os.Getenv("POSTX_DRY_RUN"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}
