package registry

import (
	"testing"
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
	reg := NewMultiAccountRegistry([]string{"scope1"})

	if !reg.IsMultiAccount() {
		t.Error("multi-account registry should be multi-account")
	}

	// Without any accounts configured, Resolve("") should fail.
	// Note: this test depends on no accounts/ dir existing in the
	// test environment. The auth.ListAccounts call will return empty
	// since BaseDir defaults to the home dir, but we can't guarantee
	// this in all environments. This is more of a smoke test.
	t.Run("EmptyAccountNoAccounts", func(t *testing.T) {
		_, err := reg.Resolve("")
		if err == nil {
			t.Error("expected error when no accounts configured and account param empty")
		}
	})
}
