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

func XBackend() string {
	var v string
	if ev := strings.TrimSpace(os.Getenv("POSTX_X_BACKEND")); ev != "" {
		v = ev
	} else if kv, err := config.LoadEnvMap(); err == nil {
		v = strings.TrimSpace(kv["POSTX_X_BACKEND"])
	}
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "x"
	}
	return v
}

func XquikAPIKey() string {
	if v := strings.TrimSpace(os.Getenv("POSTX_XQUIK_API_KEY")); v != "" {
		return v
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		return strings.TrimSpace(kv["POSTX_XQUIK_API_KEY"])
	}
	return ""
}

func XquikAccount() string {
	if v := strings.TrimSpace(os.Getenv("POSTX_XQUIK_ACCOUNT")); v != "" {
		return v
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		return strings.TrimSpace(kv["POSTX_XQUIK_ACCOUNT"])
	}
	return ""
}

func XquikCreateTweetURL() string {
	if v := strings.TrimSpace(os.Getenv("POSTX_XQUIK_CREATE_TWEET_URL")); v != "" {
		return v
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		return strings.TrimSpace(kv["POSTX_XQUIK_CREATE_TWEET_URL"])
	}
	return ""
}

func SubstackCookie() string {
	if v := strings.TrimSpace(os.Getenv("POSTX_SUBSTACK_COOKIE")); v != "" {
		return v
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		return strings.TrimSpace(kv["POSTX_SUBSTACK_COOKIE"])
	}
	return ""
}

func SubstackPublication() string {
	if v := strings.TrimSpace(os.Getenv("POSTX_SUBSTACK_PUBLICATION")); v != "" {
		return v
	}
	if kv, err := config.LoadEnvMap(); err == nil {
		return strings.TrimSpace(kv["POSTX_SUBSTACK_PUBLICATION"])
	}
	return ""
}

func SubstackSendEmail() bool {
	var v string
	if ev := strings.TrimSpace(os.Getenv("POSTX_SUBSTACK_SEND_EMAIL")); ev != "" {
		v = ev
	} else if kv, err := config.LoadEnvMap(); err == nil {
		v = strings.TrimSpace(kv["POSTX_SUBSTACK_SEND_EMAIL"])
	}
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func DryRun() bool {
	v := strings.TrimSpace(os.Getenv("POSTX_DRY_RUN"))
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}
