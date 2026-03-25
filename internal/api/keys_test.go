package api

import (
	"strings"
	"testing"
)

func TestKeyStoreIssueAndValidate(t *testing.T) {
	ks := newTestKeyStore(t)

	key, err := ks.IssueKey("test-user")
	if err != nil {
		t.Fatalf("IssueKey: %v", err)
	}
	if !strings.HasPrefix(key, "tf_") {
		t.Errorf("key should start with tf_, got %q", key)
	}

	valid, err := ks.ValidateKey(key)
	if err != nil {
		t.Fatalf("ValidateKey: %v", err)
	}
	if !valid {
		t.Error("expected key to be valid")
	}
}

func TestKeyStoreRevoke(t *testing.T) {
	ks := newTestKeyStore(t)

	key, _ := ks.IssueKey("to-revoke")
	if err := ks.RevokeKey(key); err != nil {
		t.Fatalf("RevokeKey: %v", err)
	}

	valid, _ := ks.ValidateKey(key)
	if valid {
		t.Error("expected revoked key to be invalid")
	}
}

func TestKeyStoreInvalidKey(t *testing.T) {
	ks := newTestKeyStore(t)
	valid, err := ks.ValidateKey("tf_doesnotexist")
	if err != nil {
		t.Fatalf("ValidateKey: %v", err)
	}
	if valid {
		t.Error("expected non-existent key to be invalid")
	}
}

func TestKeyStoreWALMode(t *testing.T) {
	// WAL mode is verified during NewKeyStore — if this test runs, WAL was enabled
	newTestKeyStore(t)
}

// newTestKeyStore creates a temp-file SQLite key store for testing.
// In-memory (:memory:) SQLite does not support WAL mode, so we use a temp file.
func newTestKeyStore(t *testing.T) *KeyStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	ks, err := NewKeyStore(dbPath)
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	t.Cleanup(func() { ks.Close() })
	return ks
}
