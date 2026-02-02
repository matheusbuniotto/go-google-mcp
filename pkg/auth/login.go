package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// generateStateToken generates a random state token for CSRF protection.
func generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Login performs the User OAuth 2.0 flow.
// It requires clientSecrets (content of the JSON file) and scopes.
func Login(ctx context.Context, clientSecrets []byte, scopes []string) error {
	config, err := google.ConfigFromJSON(clientSecrets, scopes...)
	if err != nil {
		return fmt.Errorf("unable to parse client secret file to config: %w", err)
	}

	// Use a fixed redirect URL for localhost
	config.RedirectURL = "http://localhost:8085/callback"

	stateToken, err := generateStateToken()
	if err != nil {
		return fmt.Errorf("failed to generate state token: %w", err)
	}

	// Create a channel to signal when the token is received
	codeChan := make(chan string)
	errChan := make(chan error)

	// Bind strictly to 127.0.0.1 to prevent network exposure
	server := &http.Server{Addr: "127.0.0.1:8085"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify state token to prevent CSRF
		if r.URL.Query().Get("state") != stateToken {
			err := fmt.Errorf("state token mismatch")
			http.Error(w, "State token mismatch", http.StatusBadRequest)
			errChan <- err
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			err := fmt.Errorf("code not found in URL")
			_, _ = fmt.Fprintf(w, "Error: %s", err)
			errChan <- err
			return
		}
		_, _ = fmt.Fprintf(w, "Success! You can close this window now.")
		codeChan <- code
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	authURL := config.AuthCodeURL(stateToken, oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)
	fmt.Println("Waiting for authentication...")

	// Open browser if possible (user task, but we print link)
	// You might use "pkg/browser" but printing is safe fallback.

	var authCode string
	select {
	case authCode = <-codeChan:
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}

	// Exchange code for token
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		return fmt.Errorf("unable to retrieve token from web: %w", err)
	}

	// Save token
	if err := SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("Authentication successful! Token saved.")
	// Shutdown server
	_ = server.Shutdown(ctx)
	return nil
}
