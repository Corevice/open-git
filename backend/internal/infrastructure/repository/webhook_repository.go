package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/infrastructure/crypto"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

type sqlxWebhookRepository struct {
	*sqlx.DB
	enc *crypto.SecretEncryptor
}

func NewWebhookRepository(db *sqlx.DB, enc *crypto.SecretEncryptor) *sqlxWebhookRepository {
	return &sqlxWebhookRepository{DB: db, enc: enc}
}

const webhookSelectColumns = `
	id, organization_id, repository_id, url, content_type, secret_encrypted, events, active, created_at, updated_at
`

func (r *sqlxWebhookRepository) Create(ctx context.Context, webhook *entity.Webhook) error {
	if webhook.ID == uuid.Nil {
		webhook.ID = uuid.New()
	}
	now := time.Now().UTC()
	if webhook.CreatedAt.IsZero() {
		webhook.CreatedAt = now
	}
	if webhook.UpdatedAt.IsZero() {
		webhook.UpdatedAt = now
	}

	secretEncrypted, err := r.encryptSecret(webhook.SecretEncrypted)
	if err != nil {
		return err
	}

	events, err := marshalWebhookEvents(r.DriverName(), webhook.Events)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO webhooks (
			id, organization_id, repository_id, url, content_type, secret_encrypted, events, active, created_at, updated_at
		) VALUES (
			:id, :organization_id, :repository_id, :url, :content_type, :secret_encrypted, :events, :active, :created_at, :updated_at
		)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":               webhook.ID,
		"organization_id":  webhook.OrganizationID,
		"repository_id":    webhook.RepositoryID,
		"url":              webhook.URL,
		"content_type":     webhook.ContentType,
		"secret_encrypted": secretEncrypted,
		"events":           events,
		"active":           webhook.Active,
		"created_at":       webhook.CreatedAt,
		"updated_at":       webhook.UpdatedAt,
	})
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	webhook.SecretEncrypted = secretEncrypted
	return nil
}

func (r *sqlxWebhookRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*entity.Webhook, error) {
	query := `SELECT ` + webhookSelectColumns + ` FROM webhooks WHERE id = ? AND organization_id = ?`
	query = r.DB.Rebind(query)

	webhook, err := r.scanWebhook(r.DB.QueryRowxContext(ctx, query, id, orgID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return webhook, nil
}

func (r *sqlxWebhookRepository) ListByRepo(ctx context.Context, orgID, repoID uuid.UUID, page, perPage int) ([]*entity.Webhook, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	countQuery := `
		SELECT COUNT(*)
		FROM webhooks
		WHERE organization_id = ? AND repository_id = ?
	`
	countQuery = r.DB.Rebind(countQuery)

	var total int64
	if err := r.DB.GetContext(ctx, &total, countQuery, orgID, repoID); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}

	query := `
		SELECT ` + webhookSelectColumns + `
		FROM webhooks
		WHERE organization_id = ? AND repository_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, orgID, repoID, perPage, offset)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	webhooks, err := r.scanWebhookRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return webhooks, total, nil
}

func (r *sqlxWebhookRepository) ListByOrg(ctx context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.Webhook, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	countQuery := `
		SELECT COUNT(*)
		FROM webhooks
		WHERE organization_id = ? AND repository_id IS NULL
	`
	countQuery = r.DB.Rebind(countQuery)

	var total int64
	if err := r.DB.GetContext(ctx, &total, countQuery, orgID); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}

	query := `
		SELECT ` + webhookSelectColumns + `
		FROM webhooks
		WHERE organization_id = ? AND repository_id IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, orgID, perPage, offset)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	webhooks, err := r.scanWebhookRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return webhooks, total, nil
}

func (r *sqlxWebhookRepository) Update(ctx context.Context, webhook *entity.Webhook) error {
	webhook.UpdatedAt = time.Now().UTC()

	secretEncrypted, err := r.encryptSecret(webhook.SecretEncrypted)
	if err != nil {
		return err
	}

	events, err := marshalWebhookEvents(r.DriverName(), webhook.Events)
	if err != nil {
		return err
	}

	const query = `
		UPDATE webhooks
		SET url = :url,
			content_type = :content_type,
			secret_encrypted = :secret_encrypted,
			events = :events,
			active = :active,
			updated_at = :updated_at
		WHERE id = :id AND organization_id = :organization_id
	`

	result, err := r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":               webhook.ID,
		"organization_id":  webhook.OrganizationID,
		"url":              webhook.URL,
		"content_type":     webhook.ContentType,
		"secret_encrypted": secretEncrypted,
		"events":           events,
		"active":           webhook.Active,
		"updated_at":       webhook.UpdatedAt,
	})
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return apperror.ErrNotFound
	}

	webhook.SecretEncrypted = secretEncrypted
	return nil
}

func (r *sqlxWebhookRepository) Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = ? AND organization_id = ?`
	query = r.DB.Rebind(query)

	result, err := r.DB.ExecContext(ctx, query, id, orgID)
	if err != nil {
		return dbErrors.MapDBError(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return dbErrors.MapDBError(err)
	}
	if rows == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *sqlxWebhookRepository) ListActiveByRepoAndEvent(ctx context.Context, orgID, repoID uuid.UUID, event string) ([]*entity.Webhook, error) {
	var (
		query string
		args  []any
	)

	if r.DriverName() == "postgres" {
		query = `
			SELECT ` + webhookSelectColumns + `
			FROM webhooks
			WHERE active = true
				AND organization_id = $1
				AND repository_id = $2
				AND (events @> $3::jsonb OR events @> $4::jsonb)
			ORDER BY created_at ASC
		`
		wildcard, err := json.Marshal([]string{"*"})
		if err != nil {
			return nil, err
		}
		specific, err := json.Marshal([]string{event})
		if err != nil {
			return nil, err
		}
		args = []any{orgID, repoID, string(wildcard), string(specific)}
	} else {
		query = `
			SELECT ` + webhookSelectColumns + `
			FROM webhooks
			WHERE active = 1
				AND organization_id = ?
				AND repository_id = ?
				AND (events LIKE ? OR events LIKE ?)
			ORDER BY created_at ASC
		`
		args = []any{orgID, repoID, `%"*"%`, fmt.Sprintf(`%%"%s"%%`, event)}
	}

	query = r.DB.Rebind(query)
	rows, err := r.DB.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	return r.scanWebhookRows(rows)
}

func (r *sqlxWebhookRepository) encryptSecret(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}
	return r.enc.Encrypt(plaintext)
}

func (r *sqlxWebhookRepository) scanWebhook(row *sqlx.Row) (*entity.Webhook, error) {
	var (
		webhook           entity.Webhook
		repositoryID      sql.NullString
		secretEncrypted   []byte
		eventsRaw         any
		updatedAt         sql.NullTime
	)

	err := row.Scan(
		&webhook.ID,
		&webhook.OrganizationID,
		&repositoryID,
		&webhook.URL,
		&webhook.ContentType,
		&secretEncrypted,
		&eventsRaw,
		&webhook.Active,
		&webhook.CreatedAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if repositoryID.Valid {
		parsed, err := uuid.Parse(repositoryID.String)
		if err != nil {
			return nil, err
		}
		webhook.RepositoryID = &parsed
	}
	if len(secretEncrypted) > 0 {
		webhook.SecretEncrypted = secretEncrypted
	}
	if updatedAt.Valid {
		webhook.UpdatedAt = updatedAt.Time
	}

	events, err := unmarshalWebhookEvents(r.DriverName(), eventsRaw)
	if err != nil {
		return nil, err
	}
	webhook.Events = events

	return &webhook, nil
}

func (r *sqlxWebhookRepository) scanWebhookRows(rows *sqlx.Rows) ([]*entity.Webhook, error) {
	var webhooks []*entity.Webhook
	for rows.Next() {
		var (
			webhook         entity.Webhook
			repositoryID    sql.NullString
			secretEncrypted []byte
			eventsRaw       any
			updatedAt       sql.NullTime
		)

		if err := rows.Scan(
			&webhook.ID,
			&webhook.OrganizationID,
			&repositoryID,
			&webhook.URL,
			&webhook.ContentType,
			&secretEncrypted,
			&eventsRaw,
			&webhook.Active,
			&webhook.CreatedAt,
			&updatedAt,
		); err != nil {
			return nil, dbErrors.MapDBError(err)
		}

		if repositoryID.Valid {
			parsed, err := uuid.Parse(repositoryID.String)
			if err != nil {
				return nil, err
			}
			webhook.RepositoryID = &parsed
		}
		if len(secretEncrypted) > 0 {
			webhook.SecretEncrypted = secretEncrypted
		}
		if updatedAt.Valid {
			webhook.UpdatedAt = updatedAt.Time
		}

		events, err := unmarshalWebhookEvents(r.DriverName(), eventsRaw)
		if err != nil {
			return nil, err
		}
		webhook.Events = events

		webhooks = append(webhooks, &webhook)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return webhooks, nil
}

func marshalWebhookEvents(_ string, events []string) (any, error) {
	if events == nil {
		events = []string{}
	}
	data, err := json.Marshal(events)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func unmarshalWebhookEvents(_ string, raw any) ([]string, error) {
	switch value := raw.(type) {
	case nil:
		return []string{}, nil
	case []byte:
		return decodeWebhookEventsJSON(value)
	case string:
		return decodeWebhookEventsJSON([]byte(value))
	default:
		return nil, fmt.Errorf("unsupported events type %T", raw)
	}
}

func decodeWebhookEventsJSON(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	var events []string
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	if events == nil {
		return []string{}, nil
	}
	return events, nil
}
