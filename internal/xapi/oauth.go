package xapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	authURL  = "https://x.com/i/oauth2/authorize"
	tokenURL = "https://api.twitter.com/2/oauth2/token"

	httpTimeout = 45 * time.Second
)

var apiHTTPClient = &http.Client{Timeout: httpTimeout}

// DefaultScopes for posting and token refresh.
var DefaultScopes = []string{"tweet.read", "tweet.write", "users.read", "offline.access"}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	ExpiresAt    time.Time
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func pkcePair() (verifier string, challenge string, err error) {
	v, err := randomString(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(v))
	ch := base64.RawURLEncoding.EncodeToString(sum[:])
	return v, ch, nil
}

// AuthorizeURL returns the URL to open in the browser and the code_verifier to keep for exchange.
func (c OAuthConfig) AuthorizeURL() (authPage string, verifier string, state string, err error) {
	if c.ClientID == "" {
		return "", "", "", fmt.Errorf("POSTX_CLIENT_ID is required")
	}
	if c.RedirectURI == "" {
		return "", "", "", fmt.Errorf("redirect URI is required")
	}
	verifier, challenge, err := pkcePair()
	if err != nil {
		return "", "", "", err
	}
	state, err = randomString(16)
	if err != nil {
		return "", "", "", err
	}
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", c.RedirectURI)
	q.Set("scope", strings.Join(DefaultScopes, " "))
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	return authURL + "?" + q.Encode(), verifier, state, nil
}

func openBrowser(target string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", target).Start()
	case "darwin":
		return exec.Command("open", target).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	default:
		return fmt.Errorf("unsupported OS for opening browser: %s", runtime.GOOS)
	}
}

// LoginInteractive opens the system browser and runs a localhost redirect server.
// ctx should include a deadline (e.g. context.WithTimeout); otherwise a stuck browser flow waits forever.
func (c OAuthConfig) LoginInteractive(ctx context.Context, listenAddr string) (Token, error) {
	authPage, verifier, wantState, err := c.AuthorizeURL()
	if err != nil {
		return Token{}, err
	}
	codeCh := make(chan string, 1)
	errCh := make(chan error, 2)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		if q.Get("error") != "" {
			select {
			case errCh <- fmt.Errorf("oauth error: %s — %s", q.Get("error"), q.Get("error_description")):
			default:
			}
			fmt.Fprintf(w, "Authorization failed. You may close this tab.")
			return
		}
		if q.Get("state") != wantState {
			select {
			case errCh <- fmt.Errorf("invalid state"):
			default:
			}
			fmt.Fprintf(w, "Invalid state. You may close this tab.")
			return
		}
		code := q.Get("code")
		if code == "" {
			select {
			case errCh <- fmt.Errorf("missing code"):
			default:
			}
			fmt.Fprintf(w, "Missing code. You may close this tab.")
			return
		}
		codeCh <- code
		fmt.Fprintf(w, "postx: authorized. You can close this tab and return to the terminal.")
	})

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return Token{}, fmt.Errorf("listen on %s: %w (is another process using this port? try POSTX_REDIRECT_URI with a free port)", listenAddr, err)
	}
	defer ln.Close()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			select {
			case errCh <- err:
			default:
			}
		}
	}()
	defer func() {
		_ = srv.Shutdown(context.Background())
	}()

	fmt.Fprintf(os.Stderr, "postx: OAuth redirect (must match X app settings): %s\n", c.RedirectURI)
	fmt.Fprintf(os.Stderr, "postx: local server listening on %s (all interfaces; use redirect URL above in the browser)\n", listenAddr)
	fmt.Fprintf(os.Stderr, "postx: opening browser. If nothing happens, paste this URL into a browser on the same machine as this terminal:\n%s\n", authPage)
	if runtime.GOOS == "linux" {
		fmt.Fprintf(os.Stderr, "postx: WSL note: if approval succeeds but this still waits, the browser may be sending 127.0.0.1 to Windows instead of Linux — run postx login from a terminal where localhost matches your browser, or see README.\n")
	}

	if err := openBrowser(authPage); err != nil {
		_ = srv.Close()
		return Token{}, fmt.Errorf("open browser: %w (open manually: %s)", err, authPage)
	}

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return Token{}, err
	case <-ctx.Done():
		return Token{}, fmt.Errorf("%w (no callback received — check redirect URL in the X portal matches %s exactly)", ctx.Err(), c.RedirectURI)
	}

	// Don't tie token exchange to the same deadline as the browser wait.
	exCtx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	return c.Exchange(exCtx, code, verifier)
}

func (c OAuthConfig) Exchange(ctx context.Context, code, verifier string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.RedirectURI)
	form.Set("code_verifier", verifier)
	form.Set("client_id", c.ClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// X's token endpoint returns 401 "Missing valid authorization header" if this is omitted.
	// Use client_id:client_secret for confidential apps; for public PKCE apps secret may be empty.
	req.SetBasicAuth(c.ClientID, c.ClientSecret)

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Token{}, fmt.Errorf("token exchange: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var raw struct {
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Token{}, err
	}
	tok := Token{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		TokenType:    raw.TokenType,
		ExpiresIn:    raw.ExpiresIn,
	}
	if raw.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().UTC().Add(time.Duration(raw.ExpiresIn) * time.Second)
	}
	return tok, nil
}

func (c OAuthConfig) Refresh(ctx context.Context, refreshToken string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", c.ClientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.ClientID, c.ClientSecret)

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Token{}, fmt.Errorf("token refresh: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var raw struct {
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Token{}, err
	}
	tok := Token{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		TokenType:    raw.TokenType,
		ExpiresIn:    raw.ExpiresIn,
	}
	if raw.RefreshToken != "" {
		tok.RefreshToken = raw.RefreshToken
	} else {
		tok.RefreshToken = refreshToken
	}
	if raw.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().UTC().Add(time.Duration(raw.ExpiresIn) * time.Second)
	}
	return tok, nil
}
