package xquikapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostTextSendsCreateTweetRequest(t *testing.T) {
	var gotKey string
	var gotBody createTweetRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s; want POST", r.Method)
		}
		gotKey = r.Header.Get("X-API-Key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tweetId":"1234567890","success":true}`))
	}))
	defer ts.Close()

	c := &Client{
		APIKey:         "test-key",
		Account:        "@acct",
		CreateTweetURL: ts.URL,
	}
	id, err := c.PostText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("PostText returned error: %v", err)
	}
	if id != "1234567890" {
		t.Fatalf("id = %q; want 1234567890", id)
	}
	if gotKey != "test-key" {
		t.Fatalf("X-API-Key = %q; want test-key", gotKey)
	}
	if gotBody.Account != "@acct" || gotBody.Text != "hello" {
		t.Fatalf("body = %+v", gotBody)
	}
}

func TestPostTextReturnsWriteActionIDForAcceptedResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"pending","writeActionId":"42"}`))
	}))
	defer ts.Close()

	c := &Client{
		APIKey:         "test-key",
		Account:        "@acct",
		CreateTweetURL: ts.URL,
	}
	id, err := c.PostText(context.Background(), "hello")
	if err != nil {
		t.Fatalf("PostText returned error: %v", err)
	}
	if id != "xquik-write-action:42" {
		t.Fatalf("id = %q; want xquik-write-action:42", id)
	}
}

func TestPostTextRequiresCredentials(t *testing.T) {
	_, err := (&Client{}).PostText(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected missing API key error")
	}
}
