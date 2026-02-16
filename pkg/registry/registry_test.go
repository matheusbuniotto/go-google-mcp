package registry

import (
	"os"
	"testing"

	"github.com/matheusbuniotto/go-google-mcp/pkg/auth"
)

func TestNewLegacyRegistry(t *testing.T) {
	// In legacy mode, Resolve always returns the same ServiceSet.
	ss := &ServiceSet{} // empty but non-nil
	reg := NewLegacyRegistry(ss)

	if reg.IsMultiAccount() {
		t.Error("legacy registry should not be multi-account")
	}

	t.Run("EmptyAccount", func(t *testing.T) {
		got, err := reg.Resolve("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != ss {
			t.Error("expected same ServiceSet pointer")
		}
	})

	t.Run("AnyAccount", func(t *testing.T) {
		got, err := reg.Resolve("ignored@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != ss {
			t.Error("expected same ServiceSet pointer regardless of account param")
		}
	})
}

func TestNewMultiAccountRegistry(t *testing.T) {
	// Use a clean temp dir so the test doesn't depend on the real home directory.
	tmpDir, err := os.MkdirTemp("", "registry-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	origBaseDir := auth.BaseDir
	auth.BaseDir = tmpDir
	defer func() { auth.BaseDir = origBaseDir }()

	reg := NewMultiAccountRegistry([]string{"scope1"})

	if !reg.IsMultiAccount() {
		t.Error("multi-account registry should be multi-account")
	}

	// With a clean temp dir, no accounts exist. Resolve("") must fail.
	t.Run("EmptyAccountNoAccounts", func(t *testing.T) {
		_, err := reg.Resolve("")
		if err == nil {
			t.Error("expected error when no accounts configured and account param empty")
		}
	})
}
