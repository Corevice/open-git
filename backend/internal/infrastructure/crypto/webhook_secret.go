package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

const webhookSecretKeyEnv = "WEBHOOK_SECRET_KEY"

type SecretEncryptor struct {
	key []byte
}

func NewSecretEncryptor(key []byte) *SecretEncryptor {
	return &SecretEncryptor{key: key}
}

func NewSecretEncryptorFromEnv() *SecretEncryptor {
	keyHex := os.Getenv(webhookSecretKeyEnv)
	if keyHex == "" {
		log.Printf("warning: %s not set, using dev-only zero key", webhookSecretKeyEnv)
		return NewSecretEncryptor(make([]byte, aes.BlockSize))
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		log.Printf("warning: %s is not valid hex, using dev-only zero key: %v", webhookSecretKeyEnv, err)
		return NewSecretEncryptor(make([]byte, aes.BlockSize))
	}
	if len(key) != aes.BlockSize {
		log.Printf("warning: %s must be 32-byte hex (%d bytes), using dev-only zero key", webhookSecretKeyEnv, aes.BlockSize)
		return NewSecretEncryptor(make([]byte, aes.BlockSize))
	}

	return NewSecretEncryptor(key)
}

func (e *SecretEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if len(e.key) != aes.BlockSize {
		return nil, fmt.Errorf("invalid encryption key length: %d", len(e.key))
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (e *SecretEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(e.key) != aes.BlockSize {
		return nil, fmt.Errorf("invalid encryption key length: %d", len(e.key))
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, encrypted := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
