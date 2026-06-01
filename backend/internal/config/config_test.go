package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	keys := []string{"PORT", "DB_TYPE"}
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

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DBType != "sqlite" {
		t.Errorf("DBType = %q, want %q", cfg.DBType, "sqlite")
	}
}
