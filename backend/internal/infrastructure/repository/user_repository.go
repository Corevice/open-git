package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
)

type sqlxUserRepository struct {
	*sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *sqlxUserRepository {
	return &sqlxUserRepository{DB: db}
}

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
	return err
}

func (r *sqlxUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	return r.getOne(ctx, `SELECT id, login, email, password_hash, created_at FROM users WHERE id = $1`, id)
}

func (r *sqlxUserRepository) GetByLogin(ctx context.Context, login string) (*entity.User, error) {
	return r.getOne(ctx, `SELECT id, login, email, password_hash, created_at FROM users WHERE login = $1`, login)
}

func (r *sqlxUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return r.getOne(ctx, `SELECT id, login, email, password_hash, created_at FROM users WHERE email = $1`, email)
}

func (r *sqlxUserRepository) getOne(ctx context.Context, query string, arg any) (*entity.User, error) {
	row := r.DB.QueryRowxContext(ctx, query, arg)

	var u entity.User
	err := row.Scan(&u.ID, &u.Login, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
