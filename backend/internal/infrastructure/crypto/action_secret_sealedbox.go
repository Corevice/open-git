package crypto

import (
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

// sealedBoxLabel domain-separates the curve25519 keypair derivation so it can
// never collide with any other use of the symmetric key.
const sealedBoxLabel = "open-git-action-secrets-box-v1"

// sealedBoxKeypair deterministically derives the server's curve25519 keypair
// from the symmetric secret key. Determinism matters so every replica advertises
// the same public key and can open boxes sealed against any replica, and so the
// key survives restarts.
func (e *SecretEncryptor) sealedBoxKeypair() (pub, priv [32]byte) {
	seed := blake2b.Sum256(append(append([]byte{}, e.key...), []byte(sealedBoxLabel)...))
	copy(priv[:], seed[:])
	pubSlice, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err == nil {
		copy(pub[:], pubSlice)
	}
	return pub, priv
}

// PublicKeyBase64 returns the base64-encoded curve25519 public key clients use
// to seal secret values (GitHub's libsodium "sealed box" format).
func (e *SecretEncryptor) PublicKeyBase64() string {
	pub, _ := e.sealedBoxKeypair()
	return base64.StdEncoding.EncodeToString(pub[:])
}

// OpenSealedBox opens an anonymous ("sealed") box that a client encrypted to the
// server's public key, returning the plaintext secret. This is the transport
// decryption; callers re-encrypt the plaintext at rest with Encrypt.
func (e *SecretEncryptor) OpenSealedBox(ciphertext []byte) ([]byte, error) {
	pub, priv := e.sealedBoxKeypair()
	out, ok := box.OpenAnonymous(nil, ciphertext, &pub, &priv)
	if !ok {
		return nil, fmt.Errorf("sealed box could not be opened with the server key")
	}
	return out, nil
}
