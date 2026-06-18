package substackapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client posts to Substack's unofficial API endpoints using session cookies.
type Client struct {
	HTTP        *http.Client
	Cookie      string
	Publication string
	SendEmail   bool
	DryRun      bool
}

// APIError captures non-2xx responses returned by Substack.
type APIError struct {
	Op         string
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("substack: %s %s: %s", e.Op, e.Status, strings.TrimSpace(e.Body))
}

func (c *Client) effectiveHTTP() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// BaseURL returns the correct scheme and host for requests.
func (c *Client) BaseURL() string {
	pub := strings.TrimSpace(c.Publication)
	if pub == "" {
		return "https://substack.com"
	}
	// If it already has dot or looks like a full domain, use it as host
	if strings.Contains(pub, ".") {
		if !strings.HasPrefix(pub, "http://") && !strings.HasPrefix(pub, "https://") {
			return "https://" + pub
		}
		return pub
	}
	return fmt.Sprintf("https://%s.substack.com", pub)
}

// PostText creates and publishes a text-only newsletter draft.
func (c *Client) PostText(ctx context.Context, text string) (string, error) {
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "[postx dry-run] substack post: %q\n", text)
		return "dry-run-substack-id", nil
	}
	return c.createAndPublish(ctx, text, "")
}

// PostTextWithMedia uploads media, creates, and publishes a newsletter draft with the media.
func (c *Client) PostTextWithMedia(ctx context.Context, text, mediaPath string) (string, error) {
	if c.DryRun {
		fmt.Fprintf(os.Stderr, "[postx dry-run] substack with media %s: %q\n", mediaPath, text)
		return "dry-run-substack-id", nil
	}
	imageURL, err := c.UploadImage(ctx, mediaPath)
	if err != nil {
		return "", fmt.Errorf("upload image to substack: %w", err)
	}
	return c.createAndPublish(ctx, text, imageURL)
}

// UploadImage performs a multipart image upload to Substack and returns its CDN URL.
func (c *Client) UploadImage(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	apiURL := c.BaseURL() + "/api/v1/image"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	cookieVal := strings.TrimSpace(c.Cookie)
	if !strings.HasPrefix(cookieVal, "substack.sid=") && cookieVal != "" {
		cookieVal = "substack.sid=" + cookieVal
	}
	if cookieVal != "" {
		req.Header.Set("Cookie", cookieVal)
	}

	resp, err := c.effectiveHTTP().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &APIError{
			Op:         "upload image",
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
		}
	}

	var out struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("no image url returned in response")
	}
	return out.URL, nil
}

func (c *Client) createAndPublish(ctx context.Context, text, imageURL string) (string, error) {
	title, bodyText := extractTitle(text)
	proseMirrorBody := BuildProseMirrorDoc(bodyText, imageURL)

	draftID, err := c.createDraft(ctx, title, proseMirrorBody)
	if err != nil {
		return "", err
	}

	err = c.publishDraft(ctx, draftID)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", draftID), nil
}

func extractTitle(text string) (string, string) {
	text = strings.TrimSpace(text)
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || text == "" {
		return "Untitled Post " + time.Now().Format("2006-01-02 15:04"), ""
	}

	// If first line is reasonably short, use it as title and the rest as body
	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > 0 && len(firstLine) <= 80 {
		var rest []string
		for _, l := range lines[1:] {
			rest = append(rest, l)
		}
		restText := strings.TrimSpace(strings.Join(rest, "\n"))
		if restText == "" {
			return firstLine, firstLine
		}
		return firstLine, restText
	}

	// Otherwise, generate a short title and use the whole text as body
	title := firstLine
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	return title, text
}

func (c *Client) createDraft(ctx context.Context, title, proseMirrorBody string) (int64, error) {
	type DraftPayload struct {
		DraftTitle    string `json:"draft_title"`
		DraftSubtitle string `json:"draft_subtitle"`
		DraftBylines  []any  `json:"draft_bylines"`
		DraftBody     string `json:"draft_body"`
	}

	payload := DraftPayload{
		DraftTitle:    title,
		DraftSubtitle: "",
		DraftBylines:  []any{},
		DraftBody:     proseMirrorBody,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	apiURL := c.BaseURL() + "/api/v1/drafts"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	cookieVal := strings.TrimSpace(c.Cookie)
	if !strings.HasPrefix(cookieVal, "substack.sid=") && cookieVal != "" {
		cookieVal = "substack.sid=" + cookieVal
	}
	if cookieVal != "" {
		req.Header.Set("Cookie", cookieVal)
	}

	resp, err := c.effectiveHTTP().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, &APIError{
			Op:         "create draft",
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
		}
	}

	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, err
	}
	if out.ID == 0 {
		return 0, fmt.Errorf("no draft id returned in response")
	}
	return out.ID, nil
}

func (c *Client) publishDraft(ctx context.Context, draftID int64) error {
	type PublishPayload struct {
		Send         bool   `json:"send"`
		PublishWeb   bool   `json:"publish_web"`
		ShareToFeed  bool   `json:"share_to_feed"`
		DeliveryType string `json:"delivery_type"`
		ScheduleFor  any    `json:"schedule_for"`
	}

	payload := PublishPayload{
		Send:         c.SendEmail,
		PublishWeb:   true,
		ShareToFeed:  true,
		DeliveryType: "all",
		ScheduleFor:  nil,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/api/v1/drafts/%d/publish", c.BaseURL(), draftID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	cookieVal := strings.TrimSpace(c.Cookie)
	if !strings.HasPrefix(cookieVal, "substack.sid=") && cookieVal != "" {
		cookieVal = "substack.sid=" + cookieVal
	}
	if cookieVal != "" {
		req.Header.Set("Cookie", cookieVal)
	}

	resp, err := c.effectiveHTTP().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			Op:         "publish draft",
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       string(body),
		}
	}
	return nil
}

// Node represents a node in a ProseMirror document.
type Node struct {
	Type    string         `json:"type"`
	Content []Node         `json:"content,omitempty"`
	Text    string         `json:"text,omitempty"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// ProseMirrorDoc represents a ProseMirror document structure.
type ProseMirrorDoc struct {
	Type    string `json:"type"`
	Content []Node `json:"content"`
}

// BuildProseMirrorDoc converts plain text and an optional image URL into the JSON representation that Substack's ProseMirror editor expects.
func BuildProseMirrorDoc(text string, imageURL string) string {
	var contentNodes []Node
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		pNode := Node{
			Type: "paragraph",
		}
		if trimmed != "" {
			pNode.Content = []Node{
				{
					Type: "text",
					Text: trimmed,
				},
			}
		}
		contentNodes = append(contentNodes, pNode)
	}
	if imageURL != "" {
		imgNode := Node{
			Type: "image",
			Attrs: map[string]any{
				"src":    imageURL,
				"alt":    "Uploaded image",
				"width":  nil,
				"height": nil,
				"title":  nil,
			},
		}
		contentNodes = append(contentNodes, imgNode)
	}
	doc := ProseMirrorDoc{
		Type:    "doc",
		Content: contentNodes,
	}
	b, _ := json.Marshal(doc)
	return string(b)
}
