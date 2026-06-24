package xquikapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const DefaultCreateTweetURL = "https://xquik.com/api/v1/x/tweets"

// Client posts text-only X updates through the Xquik API.
type Client struct {
	HTTP           *http.Client
	APIKey         string
	Account        string
	CreateTweetURL string
	DryRun         bool
}

// APIError captures non-2xx responses returned by Xquik.
type APIError struct {
	Op         string
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.Op, e.Status, strings.TrimSpace(e.Body))
}

func (c *Client) effectiveHTTP() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) endpoint() string {
	if s := strings.TrimSpace(c.CreateTweetURL); s != "" {
		return s
	}
	return DefaultCreateTweetURL
}

func (c *Client) validateCredentials() error {
	if c.DryRun {
		return nil
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("POSTX_XQUIK_API_KEY is required")
	}
	if strings.TrimSpace(c.Account) == "" {
		return fmt.Errorf("POSTX_XQUIK_ACCOUNT is required")
	}
	return nil
}

// CheckReady validates the minimum requirements needed to post.
func (c *Client) CheckReady(ctx context.Context) error {
	_ = ctx
	return c.validateCredentials()
}

type createTweetRequest struct {
	Account string `json:"account"`
	Text    string `json:"text,omitempty"`
}

type createTweetResponse struct {
	TweetID       string `json:"tweetId"`
	Success       bool   `json:"success"`
	Error         string `json:"error"`
	Status        string `json:"status"`
	WriteActionID string `json:"writeActionId"`
}

// PostText creates a text-only tweet.
func (c *Client) PostText(ctx context.Context, text string) (string, error) {
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "[postx dry-run] xquik tweet for %s: %q\n", c.Account, text)
		return "dry-run-xquik-id", nil
	}
	if err := c.validateCredentials(); err != nil {
		return "", err
	}
	body, err := json.Marshal(createTweetRequest{
		Account: c.Account,
		Text:    text,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.APIKey)

	resp, err := c.effectiveHTTP().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &APIError{
			Op:         "create tweet",
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(raw)),
		}
	}
	var out createTweetResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.TweetID != "" {
		return out.TweetID, nil
	}
	if resp.StatusCode == http.StatusAccepted && out.WriteActionID != "" {
		return "xquik-write-action:" + out.WriteActionID, nil
	}
	if out.Error != "" {
		return "", fmt.Errorf("%s", out.Error)
	}
	if out.Status != "" {
		return "", fmt.Errorf("xquik create tweet returned status %q without a tweet ID", out.Status)
	}
	return "", fmt.Errorf("xquik create tweet returned no tweet ID")
}

// PostTextWithMedia reports the current postx media boundary for Xquik.
func (c *Client) PostTextWithMedia(ctx context.Context, text, mediaPath string) (string, error) {
	_ = ctx
	_ = text
	_ = mediaPath
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "[postx dry-run] xquik local media is not sent\n")
		return "dry-run-xquik-id", nil
	}
	return "", fmt.Errorf("Xquik backend requires public media URLs; local media files still use the default X backend")
}
