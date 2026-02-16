package auth

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/oauth2"
)

func TestMultiAccount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gogo-mcp-multi-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Override BaseDir so GetConfigDir uses our temp dir.
	origBaseDir := BaseDir
	BaseDir = tmpDir
	defer func() { BaseDir = origBaseDir }()

	// Also clear env var to avoid interference.
	origEnv := os.Getenv("GO_GOOGLE_MCP_CONFIG_DIR")
	os.Unsetenv("GO_GOOGLE_MCP_CONFIG_DIR")
	defer func() {
		if origEnv != "" {
			os.Setenv("GO_GOOGLE_MCP_CONFIG_DIR", origEnv)
		}
	}()

	t.Run("IsMultiAccount_NoDir", func(t *testing.T) {
		multi, err := IsMultiAccount()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if multi {
			t.Error("expected false when accounts/ dir doesn't exist")
		}
	})

	t.Run("IsMultiAccount_EmptyDir", func(t *testing.T) {
		configDir, _ := GetConfigDir()
		accountsDir := filepath.Join(configDir, AccountsDirName)
		if err := os.MkdirAll(accountsDir, 0700); err != nil {
			t.Fatalf("failed to create accounts dir: %v", err)
		}

		multi, err := IsMultiAccount()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if multi {
			t.Error("expected false when accounts/ dir is empty")
		}

		// Clean up for subsequent tests.
		_ = os.RemoveAll(accountsDir)
	})

	t.Run("IsMultiAccount_DirWithoutToken", func(t *testing.T) {
		configDir, _ := GetConfigDir()
		accountDir := filepath.Join(configDir, AccountsDirName, "user@example.com")
		if err := os.MkdirAll(accountDir, 0700); err != nil {
			t.Fatalf("failed to create account dir: %v", err)
		}

		multi, err := IsMultiAccount()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if multi {
			t.Error("expected false when account dir has no token.json")
		}

		_ = os.RemoveAll(filepath.Join(configDir, AccountsDirName))
	})

	t.Run("SaveAndLoadTokenForAccount", func(t *testing.T) {
		account := "test@gmail.com"
		token := &oauth2.Token{
			AccessToken: "account-test-token",
		}

		if err := SaveTokenForAccount(account, token); err != nil {
			t.Fatalf("failed to save token: %v", err)
		}

		loaded, err := LoadTokenForAccount(account)
		if err != nil {
			t.Fatalf("failed to load token: %v", err)
		}
		if loaded.AccessToken != token.AccessToken {
			t.Errorf("expected %s, got %s", token.AccessToken, loaded.AccessToken)
		}
	})

	t.Run("IsMultiAccount_WithValidAccount", func(t *testing.T) {
		// The previous test created test@gmail.com with a token.
		multi, err := IsMultiAccount()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !multi {
			t.Error("expected true when account with token.json exists")
		}
	})

	t.Run("ListAccounts", func(t *testing.T) {
		// Add a second account.
		token2 := &oauth2.Token{AccessToken: "token2"}
		if err := SaveTokenForAccount("work@company.com", token2); err != nil {
			t.Fatalf("failed to save second token: %v", err)
		}

		accounts, err := ListAccounts()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(accounts) != 2 {
			t.Fatalf("expected 2 accounts, got %d: %v", len(accounts), accounts)
		}

		// Accounts should include both.
		found := map[string]bool{}
		for _, a := range accounts {
			found[a] = true
		}
		if !found["test@gmail.com"] || !found["work@company.com"] {
			t.Errorf("expected test@gmail.com and work@company.com, got %v", accounts)
		}
	})

	t.Run("ListAccounts_NoDir", func(t *testing.T) {
		// Use a fresh temp dir with no accounts.
		freshDir, _ := os.MkdirTemp("", "gogo-mcp-fresh")
		defer func() { _ = os.RemoveAll(freshDir) }()
		BaseDir = freshDir

		accounts, err := ListAccounts()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(accounts) != 0 {
			t.Errorf("expected 0 accounts, got %d", len(accounts))
		}

		BaseDir = tmpDir // restore
	})

	t.Run("LoadSecretsForAccount_FallbackToShared", func(t *testing.T) {
		account := "fallback@example.com"
		_ = SaveTokenForAccount(account, &oauth2.Token{AccessToken: "x"})

		// Write shared secrets to root config dir.
		configDir, _ := GetConfigDir()
		sharedContent := []byte(`{"shared": true}`)
		_ = os.WriteFile(filepath.Join(configDir, SecretsFileName), sharedContent, 0600)

		loaded, err := LoadSecretsForAccount(account)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(loaded) != string(sharedContent) {
			t.Errorf("expected shared secrets, got %s", string(loaded))
		}
	})

	t.Run("LoadSecretsForAccount_PerAccountOverride", func(t *testing.T) {
		account := "override@example.com"

		// Create per-account secrets.
		accountDir, _ := GetAccountDir(account)
		perAccountContent := []byte(`{"per_account": true}`)
		_ = os.WriteFile(filepath.Join(accountDir, SecretsFileName), perAccountContent, 0600)

		loaded, err := LoadSecretsForAccount(account)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(loaded) != string(perAccountContent) {
			t.Errorf("expected per-account secrets, got %s", string(loaded))
		}
	})

	t.Run("SaveSecretsForAccount", func(t *testing.T) {
		account := "savesecrets@example.com"

		// Create a source secrets file.
		srcPath := filepath.Join(tmpDir, "test_secrets.json")
		content := []byte(`{"client_id": "test-save"}`)
		_ = os.WriteFile(srcPath, content, 0644)

		if err := SaveSecretsForAccount(account, srcPath); err != nil {
			t.Fatalf("failed to save secrets: %v", err)
		}

		// Verify it was saved.
		accountDir, _ := GetAccountDir(account)
		loaded, err := os.ReadFile(filepath.Join(accountDir, SecretsFileName))
		if err != nil {
			t.Fatalf("failed to read saved secrets: %v", err)
		}
		if string(loaded) != string(content) {
			t.Errorf("expected %s, got %s", string(content), string(loaded))
		}
	})

	t.Run("GetAccountDir_CreatesDir", func(t *testing.T) {
		account := "newaccount@example.com"
		dir, err := GetAccountDir(account)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("expected account dir to be created")
		}
	})
}
