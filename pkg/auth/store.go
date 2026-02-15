package auth

import (
	"encoding/json"
	"os"
	"path/filepath"

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
