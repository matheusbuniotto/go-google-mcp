package auth

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/oauth2"
)

func TestStore(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "gogo-mcp-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Override BaseDir
	BaseDir = tmpDir

	t.Run("GetConfigDir", func(t *testing.T) {
		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := filepath.Join(tmpDir, ConfigDirName)
		if dir != expected {
			t.Errorf("expected %s, got %s", expected, dir)
		}
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("config dir was not created")
		}
	})

	t.Run("SaveAndLoadToken", func(t *testing.T) {
		token := &oauth2.Token{
			AccessToken: "test-token",
		}
		if err := SaveToken(token); err != nil {
			t.Fatalf("failed to save token: %v", err)
		}

		loaded, err := LoadToken()
		if err != nil {
			t.Fatalf("failed to load token: %v", err)
		}
		if loaded.AccessToken != token.AccessToken {
			t.Errorf("expected %s, got %s", token.AccessToken, loaded.AccessToken)
		}
	})

	t.Run("SaveAndLoadSecrets", func(t *testing.T) {
		secretsPath := filepath.Join(tmpDir, "secrets.json")
		content := []byte(`{"client_id": "test"}`)
		if err := os.WriteFile(secretsPath, content, 0644); err != nil {
			t.Fatalf("failed to create fake secrets: %v", err)
		}

		if err := SaveSecrets(secretsPath); err != nil {
			t.Fatalf("failed to save secrets: %v", err)
		}

		loaded, err := LoadSecrets()
		if err != nil {
			t.Fatalf("failed to load secrets: %v", err)
		}
		if string(loaded) != string(content) {
			t.Errorf("expected %s, got %s", string(content), string(loaded))
		}
	})
}
