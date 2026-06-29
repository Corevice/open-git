package logger

import (
	"net/http"
	"testing"
)

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
		want string
	}{
		{
			name: "authorization header masked",
			key:  "Authorization",
			val:  "Bearer abc123",
			want: "***",
		},
		{
			name: "x-auth-token masked",
			key:  "X-Auth-Token",
			val:  "secret-token",
			want: "***",
		},
		{
			name: "content-type passes through",
			key:  "Content-Type",
			val:  "application/json",
			want: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskValue(tt.key, tt.val)
			if got != tt.want {
				t.Fatalf("MaskValue(%q, %q) = %q, want %q", tt.key, tt.val, got, tt.want)
			}
		})
	}
}

func TestMaskHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer abc123")
	h.Set("X-Auth-Token", "tok-secret")
	h.Set("Content-Type", "application/json")

	masked := MaskHeaders(h)

	if masked["Authorization"] != "***" {
		t.Fatalf("expected Authorization masked, got %q", masked["Authorization"])
	}
	if masked["X-Auth-Token"] != "***" {
		t.Fatalf("expected X-Auth-Token masked, got %q", masked["X-Auth-Token"])
	}
	if masked["Content-Type"] != "application/json" {
		t.Fatalf("expected Content-Type unchanged, got %q", masked["Content-Type"])
	}
}

func TestMaskMap(t *testing.T) {
	input := map[string]interface{}{
		"username": "alice",
		"nested": map[string]interface{}{
			"password": "super-secret",
			"role":     "admin",
		},
	}

	masked := MaskMap(input)

	if masked["username"] != "alice" {
		t.Fatalf("expected username unchanged, got %v", masked["username"])
	}

	nested, ok := masked["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested map, got %T", masked["nested"])
	}
	if nested["password"] != "***" {
		t.Fatalf("expected nested password masked, got %v", nested["password"])
	}
	if nested["role"] != "admin" {
		t.Fatalf("expected nested role unchanged, got %v", nested["role"])
	}
}

func TestMaskingEmptyNilInputs(t *testing.T) {
	t.Run("nil header", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("MaskHeaders(nil) panicked: %v", r)
			}
		}()
		if got := MaskHeaders(nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})

	t.Run("empty header", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("MaskHeaders(empty) panicked: %v", r)
			}
		}()
		got := MaskHeaders(http.Header{})
		if len(got) != 0 {
			t.Fatalf("expected empty map, got %v", got)
		}
	})

	t.Run("nil map", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("MaskMap(nil) panicked: %v", r)
			}
		}()
		if got := MaskMap(nil); got != nil {
			t.Fatalf("expected nil, got %v", got)
		}
	})
}
