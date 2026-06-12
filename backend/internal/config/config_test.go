package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	keys := []string{"PORT", "DB_TYPE", "JWT_SECRET"}
	saved := make(map[string]string, len(keys))
	for _, key := range keys {
		saved[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if val, ok := saved[key]; ok && val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	})
	os.Setenv("JWT_SECRET", "test-secret")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DBType != "sqlite" {
		t.Errorf("DBType = %q, want %q", cfg.DBType, "sqlite")
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidatePortRange(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
	}{
		{name: "port 0", port: "0", wantErr: true},
		{name: "port 65536", port: "65536", wantErr: true},
		{name: "port 80", port: "80", wantErr: false},
		{name: "port abc", port: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Port:      tt.port,
				DBType:    "sqlite",
				JWTSecret: "test-secret",
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() = nil, want error for port %q", tt.port)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() = %v, want nil for port %q", err, tt.port)
			}
		})
	}
}

func TestValidatePostgresRequiresDSN(t *testing.T) {
	keys := []string{"DB_TYPE", "DB_DSN", "JWT_SECRET"}
	saved := make(map[string]string, len(keys))
	for _, key := range keys {
		saved[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if val, ok := saved[key]; ok && val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	})

	os.Setenv("DB_TYPE", "postgres")
	os.Setenv("JWT_SECRET", "test-secret")

	cfg := Load()
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() = nil, want error for postgres without DB_DSN")
	}
}
