package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/open-git/backend/internal/domain"
)

func MapDBError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotFound
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrConflict
		case "23503":
			return domain.ErrValidation
		}
	}

	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed") {
		return domain.ErrConflict
	}
	if strings.Contains(msg, "database is locked") {
		return domain.ErrUnavailable
	}

	return fmt.Errorf("database: %w", domain.ErrInternal)
}
