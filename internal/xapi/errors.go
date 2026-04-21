package xapi

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// UserMessage maps low-level errors to actionable, user-facing feedback.
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusPaymentRequired:
			return "X API returned 402 Payment Required. Add a payment method or upgrade your X developer plan, then try again."
		case http.StatusUnauthorized:
			return "X rejected this request as unauthorized. Run `postx login` again and verify POSTX_CLIENT_ID/POSTX_CLIENT_SECRET."
		case http.StatusForbidden:
			return "X rejected this request (403 Forbidden). Verify your app has tweet.write permission and required product access."
		case http.StatusTooManyRequests:
			return "X rate limit reached (429). Wait a bit, then retry."
		default:
			return fmt.Sprintf("X API error (%s): %s", apiErr.Status, compact(apiErr.Body))
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "You are not logged in. Run `postx login` before posting."
	}
	raw := strings.TrimSpace(err.Error())
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "postx_client_id is required"):
		return "Missing POSTX_CLIENT_ID. Set your X OAuth client ID in the environment first."
	case strings.Contains(lower, "postx_client_secret is required"):
		return "Missing POSTX_CLIENT_SECRET. Set your X OAuth client secret in the environment first."
	case strings.Contains(lower, "not logged in"):
		return "You are not logged in. Run `postx login` before posting."
	case strings.Contains(lower, "unsupported channel"), strings.Contains(lower, "integration not available"):
		return "That channel is not available in postx yet. Only X (Twitter) can publish today; others are preview-only."
	case strings.Contains(lower, "token refresh: 401"), strings.Contains(lower, "token exchange: 401"):
		return "X rejected your app credentials (401). Verify POSTX_CLIENT_ID and POSTX_CLIENT_SECRET, then run `postx login` again."
	case strings.Contains(lower, "payment required"), strings.Contains(lower, " 402 "):
		return "X API returned 402 Payment Required. Add a payment method or upgrade your X developer plan, then try again."
	}
	return raw
}

func compact(s string) string {
	s = strings.TrimSpace(s)
	if len(s) <= 240 {
		return s
	}
	return s[:237] + "..."
}
