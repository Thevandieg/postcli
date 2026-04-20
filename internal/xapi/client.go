package xapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"postcli/internal/store"
)

const tweetsEndpoint = "https://api.twitter.com/2/tweets"

// Client posts to X API v2 using OAuth2 user access token.
type Client struct {
	HTTP       *http.Client
	OAuth      OAuthConfig
	TokenStore *store.Store
	TokenPath  string
	DryRun     bool
}

func (c *Client) effectiveHTTP() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return apiHTTPClient
}

// AccessToken returns a valid access token, refreshing if needed.
func (c *Client) AccessToken(ctx context.Context) (string, error) {
	if c.DryRun {
		return "dry-run", nil
	}
	t, err := c.TokenStore.LoadOAuth(ctx)
	if err != nil {
		return "", fmt.Errorf("load oauth: %w", err)
	}
	if t.RefreshToken == "" {
		return t.AccessToken, nil
	}
	if !t.ExpiresAt.IsZero() && time.Until(t.ExpiresAt) < time.Minute {
		refreshed, err := c.OAuth.Refresh(ctx, t.RefreshToken)
		if err != nil {
			return "", err
		}
		sto := store.OAuthToken{
			AccessToken:  refreshed.AccessToken,
			RefreshToken: refreshed.RefreshToken,
			TokenType:    refreshed.TokenType,
			ExpiresAt:    refreshed.ExpiresAt,
		}
		if err := c.TokenStore.SaveOAuth(ctx, sto, c.TokenPath); err != nil {
			return "", err
		}
		return refreshed.AccessToken, nil
	}
	return t.AccessToken, nil
}

type createTweetRequest struct {
	Text   string              `json:"text,omitempty"`
	Media  *createTweetMedia   `json:"media,omitempty"`
}

type createTweetMedia struct {
	MediaIDs []string `json:"media_ids"`
}

type createTweetResponse struct {
	Data struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"data"`
	Errors []struct {
		Detail string `json:"detail"`
		Title  string `json:"title"`
	} `json:"errors"`
}

// PostText creates a text-only tweet.
func (c *Client) PostText(ctx context.Context, text string) (string, error) {
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "[postx dry-run] tweet: %q\n", text)
		return "dry-run-id", nil
	}
	body, err := json.Marshal(createTweetRequest{Text: text})
	if err != nil {
		return "", err
	}
	return c.postTweet(ctx, body)
}

// PostTextWithMedia uploads media then tweets with attachment.
func (c *Client) PostTextWithMedia(ctx context.Context, text, mediaPath string) (string, error) {
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "[postx dry-run] tweet with media %s: %q\n", mediaPath, text)
		return "dry-run-id", nil
	}
	mediaID, err := c.SimpleMediaUpload(ctx, mediaPath)
	if err != nil {
		return "", err
	}
	payload := createTweetRequest{
		Text: text,
		Media: &createTweetMedia{
			MediaIDs: []string{mediaID},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return c.postTweet(ctx, b)
}

func (c *Client) postTweet(ctx context.Context, jsonBody []byte) (string, error) {
	tok, err := c.AccessToken(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tweetsEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.effectiveHTTP().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("create tweet %s: %s", resp.Status, strings.TrimSpace(string(raw)))
	}
	var out createTweetResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Data.ID == "" && len(out.Errors) > 0 {
		return "", fmt.Errorf("%s", out.Errors[0].Detail)
	}
	return out.Data.ID, nil
}
