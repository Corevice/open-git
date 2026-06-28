package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/nacl/box"
)

const (
	actionSecretMasterKeyEnv = "ACTION_SECRET_MASTER_KEY"
	actionSecretHKDFInfo     = "open-git-action-secret-x25519"
	actionSecretHKDFSalt     = "open-git-action-secret-x25519-salt"
)

type ActionSecretEncryptor struct {
	*SecretEncryptor
	publicKey  [32]byte
	privateKey [32]byte
}

func NewActionSecretEncryptorFromEnv() (*ActionSecretEncryptor, error) {
	keyHex := os.Getenv(actionSecretMasterKeyEnv)
	if keyHex == "" {
		return nil, fmt.Errorf("%s is required", actionSecretMasterKeyEnv)
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("%s is not valid hex: %w", actionSecretMasterKeyEnv, err)
	}
	if len(key) != aes256KeySize {
		return nil, fmt.Errorf("%s must be %d-byte hex", actionSecretMasterKeyEnv, aes256KeySize)
	}

	return newActionSecretEncryptor(key)
}

func newActionSecretEncryptor(masterKey []byte) (*ActionSecretEncryptor, error) {
	publicKey, privateKey, err := deriveX25519KeyPair(masterKey)
	if err != nil {
		return nil, err
	}
	return &ActionSecretEncryptor{
		SecretEncryptor: NewSecretEncryptor(masterKey),
		publicKey:       publicKey,
		privateKey:      privateKey,
	}, nil
}

func deriveX25519KeyPair(masterKey []byte) (publicKey, privateKey [32]byte, err error) {
	reader := hkdf.New(sha256.New, masterKey, []byte(actionSecretHKDFSalt), []byte(actionSecretHKDFInfo))
	if _, err := io.ReadFull(reader, privateKey[:]); err != nil {
		return publicKey, privateKey, fmt.Errorf("derive x25519 private key: %w", err)
	}
	curve25519.ScalarBaseMult(&publicKey, &privateKey)
	return publicKey, privateKey, nil
}

func (e *ActionSecretEncryptor) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(e.publicKey[:])
}

func (e *ActionSecretEncryptor) KeyID() string {
	hash := sha256.Sum256(e.publicKey[:])
	return hex.EncodeToString(hash[:])
}

func (e *ActionSecretEncryptor) DecryptSealedBox(ciphertext []byte) ([]byte, error) {
	plaintext, ok := box.OpenAnonymous(nil, ciphertext, &e.publicKey, &e.privateKey)
	if !ok {
		return nil, errors.New("failed to decrypt sealed box")
	}
	return plaintext, nil
}
