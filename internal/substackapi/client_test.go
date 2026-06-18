package substackapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestBaseURL(t *testing.T) {
	tests := []struct {
		pub  string
		want string
	}{
		{"", "https://substack.com"},
		{"myblog", "https://myblog.substack.com"},
		{"myblog.com", "https://myblog.com"},
		{"https://custom.domain.org", "https://custom.domain.org"},
		{"http://local.dev", "http://local.dev"},
	}

	for _, tt := range tests {
		c := &Client{Publication: tt.pub}
		got := c.BaseURL()
		if got != tt.want {
			t.Errorf("BaseURL() for pub %q = %q; want %q", tt.pub, got, tt.want)
		}
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantTitle string
		wantBody  string
	}{
		{
			name:      "empty text",
			text:      "",
			wantTitle: "Untitled Post ", // will be followed by timestamp prefix
			wantBody:  "",
		},
		{
			name:      "short single line",
			text:      "My First Post",
			wantTitle: "My First Post",
			wantBody:  "My First Post",
		},
		{
			name: "multi-line with short first line",
			text: "My Amazing Title\n\nThis is the body of the post.\nIt has multiple lines.",
			wantTitle: "My Amazing Title",
			wantBody:  "This is the body of the post.\nIt has multiple lines.",
		},
		{
			name:      "long single line",
			text:      "This is an incredibly long line that we expect to be truncated because it exceeds the eighty character limit that we have set for a clean title",
			wantTitle: "This is an incredibly long line that we expect ...",
			wantBody:  "This is an incredibly long line that we expect to be truncated because it exceeds the eighty character limit that we have set for a clean title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, body := extractTitle(tt.text)
			if tt.name == "empty text" {
				if !strings.HasPrefix(title, tt.wantTitle) {
					t.Errorf("extractTitle(%q) title = %q; expected prefix %q", tt.text, title, tt.wantTitle)
				}
			} else if title != tt.wantTitle {
				t.Errorf("extractTitle(%q) title = %q; want %q", tt.text, title, tt.wantTitle)
			}
			if body != tt.wantBody {
				t.Errorf("extractTitle(%q) body = %q; want %q", tt.text, body, tt.wantBody)
			}
		})
	}
}

func TestBuildProseMirrorDoc(t *testing.T) {
	text := "Hello World\nSecond Line"
	imageURL := "https://cdn.substack.com/image.png"

	docJSON := BuildProseMirrorDoc(text, imageURL)

	var doc ProseMirrorDoc
	if err := json.Unmarshal([]byte(docJSON), &doc); err != nil {
		t.Fatalf("failed to unmarshal prose mirror doc: %v", err)
	}

	if doc.Type != "doc" {
		t.Errorf("expected doc type 'doc', got %q", doc.Type)
	}

	// We expect 3 nodes: paragraph ("Hello World"), paragraph ("Second Line"), image ("https://cdn.substack.com/image.png")
	if len(doc.Content) != 3 {
		t.Fatalf("expected 3 content nodes, got %d", len(doc.Content))
	}

	if doc.Content[0].Type != "paragraph" || len(doc.Content[0].Content) != 1 || doc.Content[0].Content[0].Text != "Hello World" {
		t.Errorf("unexpected first paragraph: %+v", doc.Content[0])
	}

	if doc.Content[1].Type != "paragraph" || len(doc.Content[1].Content) != 1 || doc.Content[1].Content[0].Text != "Second Line" {
		t.Errorf("unexpected second paragraph: %+v", doc.Content[1])
	}

	if doc.Content[2].Type != "image" || doc.Content[2].Attrs["src"] != imageURL {
		t.Errorf("unexpected image node: %+v", doc.Content[2])
	}
}

func TestClientDryRun(t *testing.T) {
	c := &Client{
		DryRun:      true,
		Publication: "test",
		Cookie:      "sid",
	}

	id, err := c.PostText(context.Background(), "hello dry run")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "dry-run-substack-id" {
		t.Errorf("expected dry-run-substack-id, got %q", id)
	}

	id, err = c.PostTextWithMedia(context.Background(), "hello dry run with media", "dummy.png")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "dry-run-substack-id" {
		t.Errorf("expected dry-run-substack-id, got %q", id)
	}
}

func TestClientLiveMock(t *testing.T) {
	mux := http.NewServeMux()
	var receivedCookie string

	mux.HandleFunc("/api/v1/image", func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Expect multipart body
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			http.Error(w, "expected multipart form", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"url":"https://substack-cdn.com/uploaded.png"}`))
	})

	mux.HandleFunc("/api/v1/drafts", func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			DraftTitle   string `json:"draft_title"`
			DraftBody    string `json:"draft_body"`
			DraftBylines []any  `json:"draft_bylines"`
		}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if payload.DraftTitle == "" || payload.DraftBody == "" {
			http.Error(w, "missing title or body", http.StatusBadRequest)
			return
		}
		if payload.DraftBylines == nil {
			http.Error(w, "missing draft_bylines array", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":778899}`))
	})

	mux.HandleFunc("/api/v1/drafts/778899/publish", func(w http.ResponseWriter, r *http.Request) {
		receivedCookie = r.Header.Get("Cookie")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Send        bool `json:"send"`
			PublishWeb  bool `json:"publish_web"`
			ShareToFeed bool `json:"share_to_feed"`
		}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true}`))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	c := &Client{
		HTTP:        ts.Client(),
		Publication: ts.URL, // base URL will resolve to ts.URL because it contains "http://"
		Cookie:      "my-secret-session-cookie",
		SendEmail:   true,
	}

	// 1. Test image upload
	tempFile, err := os.CreateTemp("", "test-img-*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Write([]byte("fake-image-bytes"))
	tempFile.Close()

	imgURL, err := c.UploadImage(context.Background(), tempFile.Name())
	if err != nil {
		t.Fatalf("UploadImage failed: %v", err)
	}
	if imgURL != "https://substack-cdn.com/uploaded.png" {
		t.Errorf("unexpected image URL: %q", imgURL)
	}
	if receivedCookie != "substack.sid=my-secret-session-cookie" {
		t.Errorf("expected cookie substack.sid=my-secret-session-cookie, got %q", receivedCookie)
	}

	// 2. Test full post drafting + publishing with media
	id, err := c.PostTextWithMedia(context.Background(), "My Newsletter Title\nThis is the post content.", tempFile.Name())
	if err != nil {
		t.Fatalf("PostTextWithMedia failed: %v", err)
	}
	if id != "778899" {
		t.Errorf("expected draft id '778899', got %q", id)
	}
}

func TestClientLiveMockError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized cookie or session expired", http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := &Client{
		HTTP:        ts.Client(),
		Publication: ts.URL,
		Cookie:      "bad-cookie",
	}

	_, err := c.PostText(context.Background(), "test text")
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}

	var apiErr *APIError
	if !testing.Short() && !strings.Contains(err.Error(), "unauthorized cookie") {
		t.Errorf("expected error containing details about cookie, got: %v", err)
	}

	if !errorsAs(err, &apiErr) {
		t.Errorf("expected error to be of type *APIError, got %T", err)
	} else {
		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status code 401, got %d", apiErr.StatusCode)
		}
	}
}

// Helper because errors.As might require imports or extra boilerplate, let's keep it direct.
func errorsAs(err error, target **APIError) bool {
	if e, ok := err.(*APIError); ok {
		*target = e
		return true
	}
	// unwrapping if wrapped
	unwrap, ok := err.(interface{ Unwrap() error })
	if ok {
		return errorsAs(unwrap.Unwrap(), target)
	}
	return false
}
