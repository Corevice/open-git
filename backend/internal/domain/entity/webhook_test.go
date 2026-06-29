package entity_test

import (
	"strings"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestWebhookValidate(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		contentType string
		events      []string
		secret      []byte
		wantErr     bool
		errContains string
	}{
		{
			name:        "accepts https url with json content type",
			url:         "https://example.com/hook",
			contentType: entity.ContentTypeJSON,
			events:      []string{"push"},
			wantErr:     false,
		},
		{
			name:        "accepts http url with form content type",
			url:         "http://example.com/hook",
			contentType: entity.ContentTypeForm,
			events:      []string{"pull_request", "issues"},
			wantErr:     false,
		},
		{
			name:        "rejects ftp url scheme",
			url:         "ftp://example.com/hook",
			contentType: entity.ContentTypeJSON,
			events:      []string{"push"},
			wantErr:     true,
			errContains: "invalid url scheme",
		},
		{
			name:        "rejects empty events",
			url:         "https://example.com/hook",
			contentType: entity.ContentTypeJSON,
			events:      []string{},
			wantErr:     true,
			errContains: "events must contain at least one entry",
		},
		{
			name:        "rejects unknown content type",
			url:         "https://example.com/hook",
			contentType: "xml",
			events:      []string{"push"},
			wantErr:     true,
			errContains: "invalid content_type",
		},
		{
			name:        "rejects secret exceeding max length",
			url:         "https://example.com/hook",
			contentType: entity.ContentTypeJSON,
			events:      []string{"push"},
			secret:      []byte(strings.Repeat("a", 257)),
			wantErr:     true,
			errContains: "secret exceeds maximum length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &entity.Webhook{
				URL:             tt.url,
				ContentType:     tt.contentType,
				Events:          tt.events,
				SecretEncrypted: tt.secret,
			}
			err := webhook.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			}
		})
	}
}
