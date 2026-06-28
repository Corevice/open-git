package logger

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"

	"github.com/open-git/backend/internal/domain"
)

var glog zerolog.Logger

func Init(w io.Writer, level string) {
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	glog = zerolog.New(w).Level(lvl).With().Timestamp().Str("service", "api").Logger()
}

func InitDefault(level string) {
	Init(os.Stdout, level)
}

func Global() zerolog.Logger {
	return glog
}

func FromContext(ctx context.Context) zerolog.Logger {
	rc, ok := domain.GetRequestContext(ctx)
	if !ok {
		return glog
	}

	l := glog.With().Str("request_id", rc.RequestID)
	if rc.ActorUserID != nil {
		l = l.Int64("user_id", *rc.ActorUserID)
	}
	if rc.OrganizationID != nil {
		l = l.Int64("organization_id", *rc.OrganizationID)
	}

	return l.Logger()
}
