package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/ssh"

	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxSSHKeyRepository struct {
	*sqlx.DB
}

func NewSSHKeyRepository(db *sqlx.DB) *sqlxSSHKeyRepository {
	return &sqlxSSHKeyRepository{DB: db}
}

func (r *sqlxSSHKeyRepository) Create(ctx context.Context, key *entity.SSHKey) error {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.PublicKey))
	if err != nil {
		return err
	}
	key.Fingerprint = ssh.FingerprintSHA256(pubKey)

	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	if key.CreatedAt.IsZero() {
		key.CreatedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO ssh_keys (id, user_id, title, fingerprint, public_key, created_at)
		VALUES (:id, :user_id, :title, :fingerprint, :public_key, :created_at)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":          key.ID,
		"user_id":     key.UserID,
		"title":       key.Title,
		"fingerprint": key.Fingerprint,
		"public_key":  key.PublicKey,
		"created_at":  key.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxSSHKeyRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*entity.SSHKey, error) {
	const query = `SELECT id, user_id, title, fingerprint, public_key, created_at FROM ssh_keys WHERE fingerprint = $1`

	row := r.DB.QueryRowxContext(ctx, query, fingerprint)

	var key entity.SSHKey
	err := row.Scan(&key.ID, &key.UserID, &key.Title, &key.Fingerprint, &key.PublicKey, &key.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *sqlxSSHKeyRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.SSHKey, error) {
	const query = `
		SELECT id, user_id, title, fingerprint, public_key, created_at
		FROM ssh_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.DB.QueryxContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]*entity.SSHKey, 0)
	for rows.Next() {
		var key entity.SSHKey
		if err := rows.Scan(&key.ID, &key.UserID, &key.Title, &key.Fingerprint, &key.PublicKey, &key.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, &key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *sqlxSSHKeyRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	const query = `DELETE FROM ssh_keys WHERE id = $1 AND user_id = $2`
	_, err := r.DB.ExecContext(ctx, query, id, userID)
	return err
}
