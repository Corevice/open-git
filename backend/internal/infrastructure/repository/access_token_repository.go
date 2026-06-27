package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/open-git/backend/internal/domain"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
	appmiddleware "github.com/open-git/backend/internal/middleware"
	repo "github.com/open-git/backend/internal/repository"
)

type sqlxAccessTokenRepository struct {
	*sqlx.DB
}

func NewAccessTokenRepository(db *sqlx.DB) *sqlxAccessTokenRepository {
	return &sqlxAccessTokenRepository{DB: db}
}

var _ repo.IAccessTokenRepository = (*sqlxAccessTokenRepository)(nil)

func (r *sqlxAccessTokenRepository) Create(ctx context.Context, token *domain.AccessToken) error {
	if token.ID == 0 {
		id, err := randomTokenID()
		if err != nil {
			return err
		}
		token.ID = id
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}

	scopes, err := marshalScopes(r.DriverName(), token.Scopes)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO access_tokens (id, user_id, token_hash, scopes, expires_at, revoked_at, created_at)
		VALUES (:id, :user_id, :token_hash, :scopes, :expires_at, :revoked_at, :created_at)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":         formatTokenID(token.ID),
		"user_id":    formatTokenID(token.UserID),
		"token_hash": token.TokenHash,
		"scopes":     scopes,
		"expires_at": token.ExpiresAt,
		"revoked_at": token.RevokedAt,
		"created_at": token.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxAccessTokenRepository) ListByUserID(ctx context.Context, userID int64) ([]*domain.AccessToken, error) {
	query := `
		SELECT id, user_id, token_hash, scopes, expires_at, revoked_at, created_at
		FROM access_tokens
		WHERE user_id = ? AND revoked_at IS NULL
		ORDER BY created_at DESC
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, formatTokenID(userID))
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	tokens := make([]*domain.AccessToken, 0)
	for rows.Next() {
		token, err := scanAccessToken(rows, r.DriverName())
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return tokens, nil
}

func (r *sqlxAccessTokenRepository) Revoke(ctx context.Context, tokenID, userID int64) error {
	now := time.Now().UTC()
	query := `
		UPDATE access_tokens
		SET revoked_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`
	query = r.DB.Rebind(query)

	result, err := r.DB.ExecContext(ctx, query, now, formatTokenID(tokenID), formatTokenID(userID))
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *sqlxAccessTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.AccessToken, error) {
	now := time.Now().UTC()
	query := `
		SELECT id, user_id, token_hash, scopes, expires_at, revoked_at, created_at
		FROM access_tokens
		WHERE token_hash = ? AND revoked_at IS NULL
			AND (expires_at IS NULL OR expires_at > ?)
	`
	query = r.DB.Rebind(query)

	row := r.DB.QueryRowxContext(ctx, query, tokenHash, now)
	token, err := scanAccessToken(row, r.DriverName())
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return token, nil
}

type accessTokenRow interface {
	Scan(dest ...any) error
}

func scanAccessToken(row accessTokenRow, driver string) (*domain.AccessToken, error) {
	var (
		idRaw     string
		userIDRaw string
		tokenHash string
		scopesRaw any
		expiresAt sql.NullTime
		revokedAt sql.NullTime
		createdAt time.Time
	)

	if err := row.Scan(&idRaw, &userIDRaw, &tokenHash, &scopesRaw, &expiresAt, &revokedAt, &createdAt); err != nil {
		return nil, err
	}

	id, err := parseTokenID(idRaw)
	if err != nil {
		return nil, err
	}
	userID, err := parseTokenID(userIDRaw)
	if err != nil {
		return nil, err
	}

	scopes, err := unmarshalScopes(driver, scopesRaw)
	if err != nil {
		return nil, err
	}

	token := &domain.AccessToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		Scopes:    scopes,
		CreatedAt: createdAt,
	}
	if expiresAt.Valid {
		t := expiresAt.Time
		token.ExpiresAt = &t
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		token.RevokedAt = &t
	}
	return token, nil
}

func marshalScopes(driver string, scopes []string) (any, error) {
	if scopes == nil {
		scopes = []string{}
	}
	if driver == "postgres" {
		return pq.Array(scopes), nil
	}
	data, err := json.Marshal(scopes)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func unmarshalScopes(_ string, raw any) ([]string, error) {
	switch value := raw.(type) {
	case nil:
		return []string{}, nil
	case []byte:
		return decodeScopesJSON(value)
	case string:
		return decodeScopesJSON([]byte(value))
	case pq.StringArray:
		return []string(value), nil
	default:
		return nil, fmt.Errorf("unsupported scopes type %T", raw)
	}
}

func decodeScopesJSON(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	var scopes []string
	if err := json.Unmarshal(data, &scopes); err != nil {
		return nil, err
	}
	if scopes == nil {
		return []string{}, nil
	}
	return scopes, nil
}

func randomTokenID() (int64, error) {
	for range 8 {
		var buf [8]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return 0, err
		}
		id := int64(binary.BigEndian.Uint64(buf[:]) & 0x7fffffffffffffff)
		if id != 0 {
			return id, nil
		}
	}
	return 0, fmt.Errorf("failed to generate token id")
}

func formatTokenID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func parseTokenID(raw string) (int64, error) {
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return id, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return 0, err
	}
	return appmiddleware.UUIDToInt64(parsed), nil
}
