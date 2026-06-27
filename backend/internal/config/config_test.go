package config

import (
	"strings"
	"testing"

	"github.com/open-git/backend/internal/infrastructure/database"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "invalid DB_TYPE",
			cfg: Config{
				DBType:    "invalid",
				Port:      "8080",
				JWTSecret: "secret",
			},
			wantErr: "DB_TYPE",
		},
		{
			name: "postgres requires DSN",
			cfg: Config{
				DBType:    "postgres",
				DBDSN:     "",
				Port:      "8080",
				JWTSecret: "secret",
			},
			wantErr: "DB_DSN",
		},
		{
			name: "sqlite allows empty DSN",
			cfg: Config{
				DBType:    "sqlite",
				DBDSN:     "",
				Port:      "8080",
				JWTSecret: "secret",
			},
		},
		{
			name: "port out of range",
			cfg: Config{
				DBType:    "sqlite",
				Port:      "99999",
				JWTSecret: "secret",
			},
			wantErr: "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Validate() error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMaskDSN(t *testing.T) {
	masked := database.MaskDSN("postgres://user:secret@host/db")
	if strings.Contains(masked, "secret") {
		t.Fatalf("MaskDSN() = %q, must not contain password", masked)
	}

	if got := database.MaskDSN(""); got != "" {
		t.Fatalf("MaskDSN(\"\") = %q, want empty string", got)
	}
}
