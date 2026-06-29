package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxUserRepository struct {
	*sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *sqlxUserRepository {
	return &sqlxUserRepository{DB: db}
}

const userSelectColumns = `id, login, email, password_hash, name, bio, avatar_url, created_at, updated_at`

func (r *sqlxUserRepository) Create(ctx context.Context, user *entity.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	const query = `
		INSERT INTO users (id, login, email, password_hash, created_at)
		VALUES (:id, :login, :email, :password_hash, :created_at)
	`

	_, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":            user.ID,
		"login":         user.Login,
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"created_at":    user.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxUserRepository) Update(ctx context.Context, user *entity.User) error {
	user.UpdatedAt = time.Now().UTC()

	const query = `
		UPDATE users
		SET name = :name, bio = :bio, avatar_url = :avatar_url, updated_at = :updated_at
		WHERE id = :id
	`

	result, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":         user.ID,
		"name":       user.Name,
		"bio":        user.Bio,
		"avatar_url": user.AvatarURL,
		"updated_at": user.UpdatedAt,
	})
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

func (r *sqlxUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	return r.getOne(ctx, `SELECT `+userSelectColumns+` FROM users WHERE id = ?`, id)
}

func (r *sqlxUserRepository) GetByLogin(ctx context.Context, login string) (*entity.User, error) {
	return r.getOne(ctx, `SELECT `+userSelectColumns+` FROM users WHERE login = ?`, login)
}

func (r *sqlxUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return r.getOne(ctx, `SELECT `+userSelectColumns+` FROM users WHERE email = ?`, email)
}

func (r *sqlxUserRepository) getOne(ctx context.Context, query string, arg any) (*entity.User, error) {
	query = r.DB.Rebind(query)
	row := r.DB.QueryRowxContext(ctx, query, arg)

	var (
		u         entity.User
		name      sql.NullString
		bio       sql.NullString
		avatarURL sql.NullString
		updatedAt sql.NullTime
	)
	err := row.Scan(
		&u.ID,
		&u.Login,
		&u.Email,
		&u.PasswordHash,
		&name,
		&bio,
		&avatarURL,
		&u.CreatedAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	if name.Valid {
		u.Name = name.String
	}
	if bio.Valid {
		u.Bio = bio.String
	}
	if avatarURL.Valid {
		u.AvatarURL = avatarURL.String
	}
	if updatedAt.Valid {
		u.UpdatedAt = updatedAt.Time
	}
	return &u, nil
}
