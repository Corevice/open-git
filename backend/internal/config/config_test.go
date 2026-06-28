package config_test

import (
	"strings"
	"testing"

	"github.com/open-git/backend/internal/config"
	"github.com/open-git/backend/internal/infrastructure/database"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr string
	}{
		{
			name: "invalid DB_TYPE",
			cfg: config.Config{
				DBType:    "invalid",
				Port:      "8080",
				JWTSecret: "secret",
			},
			wantErr: "DB_TYPE",
		},
		{
			name: "postgres requires DSN",
			cfg: config.Config{
				DBType:    "postgres",
				DBDSN:     "",
				Port:      "8080",
				JWTSecret: "secret",
			},
			wantErr: "DB_DSN",
		},
		{
			name: "sqlite allows empty DSN",
			cfg: config.Config{
				DBType:    "sqlite",
				DBDSN:     "",
				Port:      "8080",
				JWTSecret: "secret",
			},
		},
		{
			name: "port out of range",
			cfg: config.Config{
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

func TestLoadBaseURLs(t *testing.T) {
	t.Setenv("API_BASE_URL", "https://api.example.com/api/v3")
	t.Setenv("WEB_BASE_URL", "https://git.example.com")
	t.Setenv("DOCS_BASE_URL", "https://docs.example.com/rest")

	cfg := config.Load()
	if cfg.APIBaseURL != "https://api.example.com/api/v3" {
		t.Fatalf("APIBaseURL = %q, want https://api.example.com/api/v3", cfg.APIBaseURL)
	}
	if cfg.WebBaseURL != "https://git.example.com" {
		t.Fatalf("WebBaseURL = %q, want https://git.example.com", cfg.WebBaseURL)
	}
	if cfg.DocsBaseURL != "https://docs.example.com/rest" {
		t.Fatalf("DocsBaseURL = %q, want https://docs.example.com/rest", cfg.DocsBaseURL)
	}
}

func TestLoadBaseURLDefaults(t *testing.T) {
	t.Setenv("API_BASE_URL", "")
	t.Setenv("WEB_BASE_URL", "")
	t.Setenv("DOCS_BASE_URL", "")

	cfg := config.Load()
	if cfg.APIBaseURL != "http://localhost:8080/api/v3" {
		t.Fatalf("APIBaseURL = %q, want http://localhost:8080/api/v3", cfg.APIBaseURL)
	}
	if cfg.WebBaseURL != "http://localhost:8080" {
		t.Fatalf("WebBaseURL = %q, want http://localhost:8080", cfg.WebBaseURL)
	}
	if cfg.DocsBaseURL != "https://docs.github.com/rest" {
		t.Fatalf("DocsBaseURL = %q, want https://docs.github.com/rest", cfg.DocsBaseURL)
	}
}

func TestMetricsConfig(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("METRICS_ENABLED", "")
		t.Setenv("METRICS_PATH", "")
		t.Setenv("METRICS_AUTH_TOKEN", "")

		cfg := config.Load()
		if !cfg.MetricsEnabled {
			t.Fatalf("MetricsEnabled = false, want true")
		}
		if cfg.MetricsPath != "/metrics" {
			t.Fatalf("MetricsPath = %q, want /metrics", cfg.MetricsPath)
		}
		if cfg.MetricsAuthToken != "" {
			t.Fatalf("MetricsAuthToken = %q, want empty", cfg.MetricsAuthToken)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		t.Setenv("METRICS_ENABLED", "false")

		cfg := config.Load()
		if cfg.MetricsEnabled {
			t.Fatalf("MetricsEnabled = true, want false")
		}
	})

	t.Run("custom path", func(t *testing.T) {
		t.Setenv("METRICS_PATH", "/prom")

		cfg := config.Load()
		if cfg.MetricsPath != "/prom" {
			t.Fatalf("MetricsPath = %q, want /prom", cfg.MetricsPath)
		}
	})

	t.Run("auth token", func(t *testing.T) {
		t.Setenv("METRICS_AUTH_TOKEN", "tok123")

		cfg := config.Load()
		if cfg.MetricsAuthToken != "tok123" {
			t.Fatalf("MetricsAuthToken = %q, want tok123", cfg.MetricsAuthToken)
		}
	})
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
