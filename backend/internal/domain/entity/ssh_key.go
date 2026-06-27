package entity

import (
	"errors"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
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
	if k.PublicKey == "" {
		return errors.New("public key is required")
	}
	return nil
}
