package xapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

const mediaUploadURL = "https://upload.twitter.com/1.1/media/upload.json"

type mediaUploadResponse struct {
	MediaIDString string `json:"media_id_string"`
	Errors        []struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"errors"`
}

// SimpleMediaUpload performs a non-chunked upload (suitable for small images).
func (c *Client) SimpleMediaUpload(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("media", path)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	tok, err := c.AccessToken(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mediaUploadURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.effectiveHTTP().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("media upload %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out mediaUploadResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.MediaIDString == "" && len(out.Errors) > 0 {
		return "", fmt.Errorf("%s", out.Errors[0].Message)
	}
	return out.MediaIDString, nil
}
