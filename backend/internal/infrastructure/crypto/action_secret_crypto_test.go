package crypto_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"golang.org/x/crypto/nacl/box"

	"github.com/open-git/backend/internal/infrastructure/crypto"
)

func newTestActionSecretEncryptor(t *testing.T) *crypto.ActionSecretEncryptor {
	t.Helper()

	key := bytes.Repeat([]byte{0x42}, 32)
	t.Setenv("ACTION_SECRET_MASTER_KEY", hex.EncodeToString(key))
	enc, err := crypto.NewActionSecretEncryptorFromEnv()
	if err != nil {
		t.Fatalf("NewActionSecretEncryptorFromEnv: %v", err)
	}
	return enc
}

func TestActionSecretEncryptor_EncryptDecryptRoundTrip(t *testing.T) {
	enc := newTestActionSecretEncryptor(t)

	plaintext := []byte("super-secret-value")
	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestActionSecretEncryptor_KeyIDStable(t *testing.T) {
	enc := newTestActionSecretEncryptor(t)

	first := enc.KeyID()
	second := enc.KeyID()
	if first == "" {
		t.Fatal("KeyID returned empty string")
	}
	if len(first) != 64 {
		t.Fatalf("KeyID length = %d, want 64", len(first))
	}
	if first != second {
		t.Fatalf("KeyID not stable: first=%q second=%q", first, second)
	}
}

func TestActionSecretEncryptor_PublicKeyBase64NonEmpty(t *testing.T) {
	enc := newTestActionSecretEncryptor(t)

	pubKey := enc.PublicKeyBase64()
	if pubKey == "" {
		t.Fatal("PublicKeyBase64 returned empty string")
	}

	decoded, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		t.Fatalf("PublicKeyBase64 is not valid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32-byte public key, got %d bytes", len(decoded))
	}
}

func TestActionSecretEncryptor_DecryptSealedBox(t *testing.T) {
	enc := newTestActionSecretEncryptor(t)

	pubKeyBytes, err := base64.StdEncoding.DecodeString(enc.PublicKeyBase64())
	if err != nil {
		t.Fatalf("decode public key: %v", err)
	}
	var pubKey [32]byte
	copy(pubKey[:], pubKeyBytes)

	plaintext := []byte("sealed-secret")
	sealed, err := box.SealAnonymous(nil, plaintext, &pubKey, rand.Reader)
	if err != nil {
		t.Fatalf("SealAnonymous: %v", err)
	}

	got, err := enc.DecryptSealedBox(sealed)
	if err != nil {
		t.Fatalf("DecryptSealedBox: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("DecryptSealedBox mismatch: got %q, want %q", got, plaintext)
	}
}

func TestNewActionSecretEncryptorFromEnv_MissingKey(t *testing.T) {
	t.Setenv("ACTION_SECRET_MASTER_KEY", "")

	_, err := crypto.NewActionSecretEncryptorFromEnv()
	if err == nil {
		t.Fatal("expected error for missing ACTION_SECRET_MASTER_KEY")
	}
	if !strings.Contains(err.Error(), "ACTION_SECRET_MASTER_KEY is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewActionSecretEncryptorFromEnv_InvalidHex(t *testing.T) {
	t.Setenv("ACTION_SECRET_MASTER_KEY", "not-valid-hex")

	_, err := crypto.NewActionSecretEncryptorFromEnv()
	if err == nil {
		t.Fatal("expected error for invalid hex key")
	}
	if !strings.Contains(err.Error(), "not valid hex") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewActionSecretEncryptorFromEnv_ShortKey(t *testing.T) {
	t.Setenv("ACTION_SECRET_MASTER_KEY", hex.EncodeToString([]byte{0x01, 0x02}))

	_, err := crypto.NewActionSecretEncryptorFromEnv()
	if err == nil {
		t.Fatal("expected error for short key")
	}
	if !strings.Contains(err.Error(), "must be 32-byte hex") {
		t.Fatalf("unexpected error: %v", err)
	}
}
