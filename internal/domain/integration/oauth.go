package integration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// OAuthHandler handles PKCE OAuth 2.1 flows.
type OAuthHandler struct {
	config *oauth2.Config
}

func NewOAuthHandler(clientID, clientSecret, authURL, tokenURL string, scopes []string) *OAuthHandler {
	return &OAuthHandler{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
			RedirectURL: "http://localhost:6299/callback",
			Scopes:      scopes,
		},
	}
}

// GeneratePKCE creates a code verifier and challenge.
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.New()
	h.Write([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return verifier, challenge, nil
}

// Login initiates the OAuth flow and captures the token.
func (h *OAuthHandler) Login(ctx context.Context) (*oauth2.Token, error) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return nil, err
	}

	state := "random-state" // Should be dynamic in real implementation
	authURL := h.config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	fmt.Printf("Please log in at: %s\n", authURL)

	// Start temporary server for callback
	codeChan := make(chan string)
	errChan := make(chan error)
	srv := &http.Server{Addr: ":6299"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("state") != state {
			errChan <- fmt.Errorf("invalid state")
			return
		}
		code := query.Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code received")
			return
		}
		fmt.Fprintln(w, "Authentication successful! You can close this window.")
		codeChan <- code
	})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	defer srv.Shutdown(ctx)

	select {
	case code := <-codeChan:
		return h.config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("timeout")
	}
}
