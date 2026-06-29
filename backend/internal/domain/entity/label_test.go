package entity_test

import (
	"testing"

	"github.com/open-git/backend/internal/domain/entity"
)

func TestLabelValidateColor(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		wantErr bool
	}{
		{
			name:    "accepts valid hex color",
			color:   "ff0000",
			wantErr: false,
		},
		{
			name:    "rejects hash prefix",
			color:   "#ff0000",
			wantErr: true,
		},
		{
			name:    "rejects non-hex characters",
			color:   "gg0000",
			wantErr: true,
		},
		{
			name:    "rejects too short",
			color:   "ff00",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			label := &entity.Label{Color: tt.color}
			err := label.ValidateColor()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for color %q", tt.color)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error for color %q: %v", tt.color, err)
			}
		})
	}
}
