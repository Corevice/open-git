package entity_test

import (
	"strings"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestSSHKeyValidate(t *testing.T) {
	validPublicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICqnyMswoAMfK42kfx8S0b1Z1BbT4NmPzKvVJ4OEmfr user@host"

	tests := []struct {
		name        string
		title       string
		keyType     string
		fingerprint string
		publicKey   string
		wantErr     bool
	}{
		{
			name:        "valid key passes",
			title:       "a",
			keyType:     "ssh-ed25519",
			fingerprint: "SHA256:example",
			publicKey:   validPublicKey,
			wantErr:     false,
		},
		{
			name:        "255-char title passes",
			title:       strings.Repeat("a", 255),
			keyType:     "ssh-ed25519",
			fingerprint: "SHA256:example",
			publicKey:   validPublicKey,
			wantErr:     false,
		},
		{
			name:        "empty title fails",
			title:       "",
			keyType:     "ssh-ed25519",
			fingerprint: "SHA256:example",
			publicKey:   validPublicKey,
			wantErr:     true,
		},
		{
			name:        "256-char title fails",
			title:       strings.Repeat("a", 256),
			keyType:     "ssh-ed25519",
			fingerprint: "SHA256:example",
			publicKey:   validPublicKey,
			wantErr:     true,
		},
		{
			name:        "empty key type fails",
			title:       "my key",
			keyType:     "",
			fingerprint: "SHA256:example",
			publicKey:   validPublicKey,
			wantErr:     true,
		},
		{
			name:        "empty fingerprint fails",
			title:       "my key",
			keyType:     "ssh-ed25519",
			fingerprint: "",
			publicKey:   validPublicKey,
			wantErr:     true,
		},
		{
			name:        "empty public key fails",
			title:       "my key",
			keyType:     "ssh-ed25519",
			fingerprint: "SHA256:example",
			publicKey:   "",
			wantErr:     true,
		},
		{
			name:        "invalid public key format fails",
			title:       "my key",
			keyType:     "ssh-ed25519",
			fingerprint: "SHA256:example",
			publicKey:   "not-a-valid-key",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &entity.SSHKey{
				Title:       tt.title,
				KeyType:     tt.keyType,
				Fingerprint: tt.fingerprint,
				PublicKey:   tt.publicKey,
			}
			err := key.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for title=%q keyType=%q fingerprint=%q publicKey=%q", tt.title, tt.keyType, tt.fingerprint, tt.publicKey)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for title=%q keyType=%q fingerprint=%q publicKey=%q: %v", tt.title, tt.keyType, tt.fingerprint, tt.publicKey, err)
			}
		})
	}
}
