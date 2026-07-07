package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/nacl/box"
)

func TestSealedBoxRoundTrip(t *testing.T) {
	key := make([]byte, aes256KeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc := NewSecretEncryptor(key)

	// A client seals a value against the advertised public key.
	pubB64 := enc.PublicKeyBase64()
	pubBytes, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil || len(pubBytes) != 32 {
		t.Fatalf("public key not valid base64/32 bytes: %v (len %d)", err, len(pubBytes))
	}
	var pub [32]byte
	copy(pub[:], pubBytes)

	secret := []byte("super-secret-token-42")
	sealed, err := box.SealAnonymous(nil, secret, &pub, rand.Reader)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	// The server opens it.
	got, err := enc.OpenSealedBox(sealed)
	if err != nil {
		t.Fatalf("OpenSealedBox: %v", err)
	}
	if string(got) != string(secret) {
		t.Errorf("opened secret = %q, want %q", got, secret)
	}
}

func TestPublicKeyIsNotPlaceholderAndDeterministic(t *testing.T) {
	key := make([]byte, aes256KeySize) // all-zero dev key
	enc := NewSecretEncryptor(key)

	pub := enc.PublicKeyBase64()
	if pub == "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" || pub == "" {
		t.Fatalf("public key is the placeholder/empty: %q", pub)
	}
	// Deterministic across instances with the same key (needed for multi-replica).
	if again := NewSecretEncryptor(key).PublicKeyBase64(); again != pub {
		t.Errorf("public key not deterministic: %q vs %q", pub, again)
	}
}

func TestOpenSealedBoxRejectsGarbage(t *testing.T) {
	enc := NewSecretEncryptor(make([]byte, aes256KeySize))
	if _, err := enc.OpenSealedBox([]byte("not a sealed box")); err == nil {
		t.Error("expected error opening non-sealed-box input")
	}
}
