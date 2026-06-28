package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestInit_ValidLevel(t *testing.T) {
	Init("debug")

	if zerolog.GlobalLevel() != zerolog.DebugLevel {
		t.Fatalf("GlobalLevel() = %v, want %v", zerolog.GlobalLevel(), zerolog.DebugLevel)
	}

	var buf bytes.Buffer
	l := zerolog.New(&buf).With().Timestamp().Str("service", "api").Logger()
	l.Debug().Msg("test")

	if !strings.Contains(buf.String(), `"level":"debug"`) {
		t.Fatalf("output = %q, want JSON containing level debug", buf.String())
	}
}

func TestInit_InvalidLevel_FallsBackToInfo(t *testing.T) {
	Init("garbage")

	if zerolog.GlobalLevel() != zerolog.InfoLevel {
		t.Fatalf("GlobalLevel() = %v, want %v", zerolog.GlobalLevel(), zerolog.InfoLevel)
	}
}

func TestGet_ReturnsNonZeroLogger(t *testing.T) {
	Init("info")
	l := Get()
	if !l.Info().Enabled() {
		t.Fatal("Get() returned zero or disabled logger")
	}
}
