package entity

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

type SSHKey struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	KeyType     string
	PublicKey   string
	Fingerprint string
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

func (k *SSHKey) Validate() error {
	if k.Title == "" {
		return errors.New("title is required")
	}
	if utf8.RuneCountInString(k.Title) > 255 {
		return errors.New("title must be at most 255 characters")
	}
	if k.KeyType == "" {
		return errors.New("key type is required")
	}
	if k.Fingerprint == "" {
		return errors.New("fingerprint is required")
	}
	if k.PublicKey == "" {
		return errors.New("public key is required")
	}
	keyLine := strings.TrimSpace(k.PublicKey)
	if !strings.Contains(keyLine, " ") {
		return errors.New("invalid public key format")
	}
	if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyLine)); err != nil {
		return errors.New("invalid public key format")
	}
	return nil
}
