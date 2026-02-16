package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
)

const (
	ConfigDirName   = ".go-google-mcp"
	TokenFileName   = "token.json"
	SecretsFileName = "client_secrets.json"
)

// BaseDir allows overriding the home directory for testing purposes.
var BaseDir string

// GetConfigDir returns the path to the configuration directory.
// Override with GO_GOOGLE_MCP_CONFIG_DIR env var for multi-instance deployments.
func GetConfigDir() (string, error) {
	if envDir := os.Getenv("GO_GOOGLE_MCP_CONFIG_DIR"); envDir != "" {
		if err := os.MkdirAll(envDir, 0700); err != nil {
			return "", err
		}
		return envDir, nil
	}

	var home string
	var err error
	if BaseDir != "" {
		home = BaseDir
	} else {
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	dir := filepath.Join(home, ConfigDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// SaveToken saves the OAuth2 token to disk.
func SaveToken(token *oauth2.Token) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, TokenFileName)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	return json.NewEncoder(f).Encode(token)
}

// LoadToken loads the OAuth2 token from disk.
func LoadToken() (*oauth2.Token, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, TokenFileName)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	var token oauth2.Token
	if err := json.NewDecoder(f).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// SaveSecrets copies the client secrets file to the config dir.
func SaveSecrets(srcPath string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	dstPath := filepath.Join(dir, SecretsFileName)

	return os.WriteFile(dstPath, content, 0600)
}

// LoadSecrets loads the client secrets from the config dir.
func LoadSecrets() ([]byte, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, SecretsFileName)
	return os.ReadFile(path)
}

const AccountsDirName = "accounts"

// validateAccountName rejects account names that could escape the
// accounts/ directory (path traversal) or cause filesystem issues.
func validateAccountName(account string) error {
	if account == "" {
		return fmt.Errorf("account name cannot be empty")
	}
	if strings.Contains(account, "..") || strings.Contains(account, "/") || strings.Contains(account, "\\") || strings.Contains(account, "\x00") {
		return fmt.Errorf("invalid account name %q: must not contain path separators or '..'", account)
	}
	return nil
}

// IsMultiAccount returns true if the accounts/ subdirectory exists
// and contains at least one account directory with a valid token.json.
func IsMultiAccount() (bool, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return false, err
	}
	accountsDir := filepath.Join(dir, AccountsDirName)
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() {
			tokenPath := filepath.Join(accountsDir, e.Name(), TokenFileName)
			if _, err := os.Stat(tokenPath); err == nil {
				return true, nil
			}
		}
	}
	return false, nil
}

// ListAccounts returns the names of all configured accounts
// (directory names under accounts/ that contain a valid token.json).
func ListAccounts() ([]string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}
	accountsDir := filepath.Join(dir, AccountsDirName)
	entries, err := os.ReadDir(accountsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var accounts []string
	for _, e := range entries {
		if e.IsDir() {
			tokenPath := filepath.Join(accountsDir, e.Name(), TokenFileName)
			if _, err := os.Stat(tokenPath); err == nil {
				accounts = append(accounts, e.Name())
			}
		}
	}
	return accounts, nil
}

// GetAccountDir returns the config directory for a specific account.
// Creates it if it doesn't exist. The account name is validated to
// prevent path traversal.
func GetAccountDir(account string) (string, error) {
	if err := validateAccountName(account); err != nil {
		return "", err
	}
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	accountDir := filepath.Join(dir, AccountsDirName, account)
	if err := os.MkdirAll(accountDir, 0700); err != nil {
		return "", err
	}
	return accountDir, nil
}

// SaveTokenForAccount saves an OAuth2 token for a specific account.
func SaveTokenForAccount(account string, token *oauth2.Token) error {
	dir, err := GetAccountDir(account)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, TokenFileName)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	return json.NewEncoder(f).Encode(token)
}

// LoadTokenForAccount loads the OAuth2 token for a specific account.
func LoadTokenForAccount(account string) (*oauth2.Token, error) {
	dir, err := GetAccountDir(account)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, TokenFileName)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	var token oauth2.Token
	if err := json.NewDecoder(f).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// LoadSecretsForAccount loads client secrets for an account.
// Falls back to the shared (root-level) client_secrets.json if the
// account doesn't have its own.
func LoadSecretsForAccount(account string) ([]byte, error) {
	// Try per-account secrets first.
	dir, err := GetAccountDir(account)
	if err != nil {
		return nil, err
	}
	perAccount := filepath.Join(dir, SecretsFileName)
	if data, err := os.ReadFile(perAccount); err == nil {
		return data, nil
	}
	// Fall back to shared secrets.
	return LoadSecrets()
}

// SaveSecretsForAccount copies client secrets into an account's directory.
func SaveSecretsForAccount(account string, srcPath string) error {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	dir, err := GetAccountDir(account)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, SecretsFileName), content, 0600)
}
