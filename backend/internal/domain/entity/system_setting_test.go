package entity_test

import (
	"reflect"
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestSystemSettingZeroValue(t *testing.T) {
	var s entity.SystemSetting
	if s.Key != "" {
		t.Fatalf("Key = %q, want empty string", s.Key)
	}
}

func TestSystemSettingNestedValue(t *testing.T) {
	s := entity.SystemSetting{
		Value: map[string]any{
			"nested": map[string]any{
				"count": float64(42),
			},
		},
	}
	nested, ok := s.Value["nested"].(map[string]any)
	if !ok {
		t.Fatal("expected nested map in Value")
	}
	if nested["count"] != float64(42) {
		t.Fatalf("nested count = %v, want 42", nested["count"])
	}
}

func TestSystemSettingStructTags(t *testing.T) {
	typ := reflect.TypeOf(entity.SystemSetting{})
	tests := []struct {
		field   string
		dbTag   string
		jsonTag string
	}{
		{"Key", "key", "key"},
		{"Value", "value", "value"},
		{"UpdatedBy", "updated_by", "updated_by"},
		{"UpdatedAt", "updated_at", "updated_at"},
	}
	for _, tt := range tests {
		field, ok := typ.FieldByName(tt.field)
		if !ok {
			t.Fatalf("field %q not found", tt.field)
		}
		if got := field.Tag.Get("db"); got != tt.dbTag {
			t.Fatalf("%s db tag = %q, want %q", tt.field, got, tt.dbTag)
		}
		if got := field.Tag.Get("json"); got != tt.jsonTag {
			t.Fatalf("%s json tag = %q, want %q", tt.field, got, tt.jsonTag)
		}
	}
}
