package logger

import (
	"os"

	"github.com/rs/zerolog"
)

var glog zerolog.Logger

func Init(level string) {
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		parsed = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(parsed)
	glog = zerolog.New(os.Stdout).With().Timestamp().Str("service", "api").Logger()
}

func Get() zerolog.Logger {
	return glog
}
