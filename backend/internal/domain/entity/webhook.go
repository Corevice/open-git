package entity

import (
	"errors"
	"net/url"
	"time"

	"github.com/google/uuid"
)

const (
	ContentTypeJSON = "json"
	ContentTypeForm = "form"

	maxWebhookSecretLength = 256
)

type Webhook struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID
	RepositoryID    *uuid.UUID
	URL             string
	ContentType     string
	SecretEncrypted []byte
	Events          []string
	Active          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (w *Webhook) Validate() error {
	parsedURL, err := url.Parse(w.URL)
	if err != nil || parsedURL.Scheme == "" {
		return errors.New("invalid url: must be a valid http or https URL")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.New("invalid url scheme: only http and https are allowed")
	}

	switch w.ContentType {
	case ContentTypeJSON, ContentTypeForm:
	default:
		return errors.New("invalid content_type: must be json or form")
	}

	if len(w.Events) < 1 {
		return errors.New("events must contain at least one entry")
	}

	if len(w.SecretEncrypted) > maxWebhookSecretLength {
		return errors.New("secret exceeds maximum length of 256 bytes")
	}

	return nil
}
