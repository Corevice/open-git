package crypto_test

import (
	"bytes"
	"testing"

	"github.com/open-git/backend/internal/infrastructure/crypto"
)

func TestSecretEncryptor_RoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0xAB}, 32)
	enc := crypto.NewSecretEncryptor(key)

	plaintext := []byte("my-webhook-secret")
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

func TestSecretEncryptor_DifferentCiphertextForSamePlaintext(t *testing.T) {
	key := bytes.Repeat([]byte{0xCD}, 32)
	enc := crypto.NewSecretEncryptor(key)

	plaintext := []byte("same-secret")
	first, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("first Encrypt: %v", err)
	}
	second, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("second Encrypt: %v", err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("expected different ciphertexts for repeated encryption of same plaintext")
	}
}

func TestSecretEncryptor_DecryptCorruptedCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0xEF}, 32)
	enc := crypto.NewSecretEncryptor(key)

	ciphertext, err := enc.Encrypt([]byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	ciphertext[len(ciphertext)-1] ^= 0xFF
	if _, err := enc.Decrypt(ciphertext); err == nil {
		t.Fatal("expected error decrypting corrupted ciphertext")
	}
}
