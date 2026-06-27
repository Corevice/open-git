package entity_test

import (
	"strings"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestSSHKeyValidate(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		publicKey string
		wantErr   bool
	}{
		{
			name:      "valid key passes",
			title:     "a",
			publicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey user@host",
			wantErr:   false,
		},
		{
			name:      "empty title fails",
			title:     "",
			publicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey user@host",
			wantErr:   true,
		},
		{
			name:      "256-char title fails",
			title:     strings.Repeat("a", 256),
			publicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKey user@host",
			wantErr:   true,
		},
		{
			name:      "empty public key fails",
			title:     "my key",
			publicKey: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &entity.SSHKey{
				Title:     tt.title,
				PublicKey: tt.publicKey,
			}
			err := key.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for title=%q publicKey=%q", tt.title, tt.publicKey)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for title=%q publicKey=%q: %v", tt.title, tt.publicKey, err)
			}
		})
	}
}

func TestHostKeyZeroValue(t *testing.T) {
	var key entity.HostKey
	if key.ID.String() != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("expected zero UUID, got %s", key.ID)
	}
	if key.Algorithm != "" {
		t.Fatalf("expected empty algorithm, got %q", key.Algorithm)
	}
	if key.PrivateKey != "" {
		t.Fatalf("expected empty private key, got %q", key.PrivateKey)
	}
	if !key.CreatedAt.IsZero() {
		t.Fatalf("expected zero CreatedAt, got %v", key.CreatedAt)
	}
}
