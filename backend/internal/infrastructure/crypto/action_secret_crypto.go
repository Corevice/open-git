package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	actionSecretMasterKeyEnv = "ACTION_SECRET_MASTER_KEY"
	actionSecretHKDFInfo     = "open-git-action-secret-x25519"
)

type ActionSecretEncryptor struct {
	*SecretEncryptor
	publicKey  [32]byte
	privateKey [32]byte
}

func NewActionSecretEncryptorFromEnv() *ActionSecretEncryptor {
	keyHex := os.Getenv(actionSecretMasterKeyEnv)
	if keyHex == "" {
		log.Printf("warning: %s not set, using dev-only zero key", actionSecretMasterKeyEnv)
		return newActionSecretEncryptor(make([]byte, aes256KeySize))
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		log.Printf("warning: %s is not valid hex, using dev-only zero key: %v", actionSecretMasterKeyEnv, err)
		return newActionSecretEncryptor(make([]byte, aes256KeySize))
	}
	if len(key) != aes256KeySize {
		log.Printf("warning: %s must be 32-byte hex (%d bytes), using dev-only zero key", actionSecretMasterKeyEnv, aes256KeySize)
		return newActionSecretEncryptor(make([]byte, aes256KeySize))
	}

	return newActionSecretEncryptor(key)
}

func newActionSecretEncryptor(masterKey []byte) *ActionSecretEncryptor {
	publicKey, privateKey := deriveX25519KeyPair(masterKey)
	return &ActionSecretEncryptor{
		SecretEncryptor: NewSecretEncryptor(masterKey),
		publicKey:       publicKey,
		privateKey:      privateKey,
	}
}

func deriveX25519KeyPair(masterKey []byte) (publicKey, privateKey [32]byte) {
	reader := hkdf.New(sha256.New, masterKey, nil, []byte(actionSecretHKDFInfo))
	if _, err := io.ReadFull(reader, privateKey[:]); err != nil {
		return publicKey, privateKey
	}
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return publicKey, privateKey
}

func (e *ActionSecretEncryptor) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(e.publicKey[:])
}

func (e *ActionSecretEncryptor) KeyID() string {
	hash := sha256.Sum256(e.publicKey[:])
	return hex.EncodeToString(hash[:])[:16]
}

func (e *ActionSecretEncryptor) DecryptSealedBox(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 32+secretbox.Overhead {
		return nil, errors.New("ciphertext too short")
	}

	var ephemeralPublicKey [32]byte
	copy(ephemeralPublicKey[:], ciphertext[:32])

	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, &e.privateKey, &ephemeralPublicKey)

	var nonce [24]byte
	copy(nonce[:], sharedSecret[:])
	nonce[23] ^= 1

	plaintext, ok := secretbox.Open(nil, ciphertext[32:], &nonce, &sharedSecret)
	if !ok {
		return nil, errors.New("failed to decrypt sealed box")
	}
	return plaintext, nil
}
