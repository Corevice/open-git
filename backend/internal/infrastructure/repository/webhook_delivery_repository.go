package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	dbErrors "github.com/open-git/backend/internal/infrastructure/database"
)

const maxWebhookDeliveryResponseBodyBytes = 64 * 1024

type sqlxWebhookDeliveryRepository struct {
	*sqlx.DB
}

func NewWebhookDeliveryRepository(db *sqlx.DB) *sqlxWebhookDeliveryRepository {
	return &sqlxWebhookDeliveryRepository{DB: db}
}

const webhookDeliverySelectColumns = `
	id, webhook_id, organization_id, event, status, status_code,
	request_headers, request_body, response_headers, response_body,
	duration_ms, attempt, redelivery, parent_delivery_id, delivered_at, created_at
`

func (r *sqlxWebhookDeliveryRepository) Create(ctx context.Context, delivery *entity.WebhookDelivery) error {
	if delivery.ID == uuid.Nil {
		delivery.ID = uuid.New()
	}
	if delivery.CreatedAt.IsZero() {
		delivery.CreatedAt = time.Now().UTC()
	}

	requestHeaders, err := marshalHeaderMap(delivery.RequestHeaders)
	if err != nil {
		return err
	}

	const query = `
		INSERT INTO webhook_deliveries (
			id, webhook_id, organization_id, event, status, status_code,
			request_headers, request_body, response_headers, response_body,
			duration_ms, attempt, redelivery, parent_delivery_id, delivered_at, created_at
		) VALUES (
			:id, :webhook_id, :organization_id, :event, :status, :status_code,
			:request_headers, :request_body, :response_headers, :response_body,
			:duration_ms, :attempt, :redelivery, :parent_delivery_id, :delivered_at, :created_at
		)
	`

	_, err = r.DB.NamedExecContext(ctx, query, map[string]any{
		"id":                 delivery.ID,
		"webhook_id":         delivery.WebhookID,
		"organization_id":    delivery.OrganizationID,
		"event":              delivery.Event,
		"status":             delivery.Status,
		"status_code":        delivery.StatusCode,
		"request_headers":    requestHeaders,
		"request_body":       delivery.RequestBody,
		"response_headers":   nil,
		"response_body":      delivery.ResponseBody,
		"duration_ms":        delivery.DurationMs,
		"attempt":            delivery.Attempt,
		"redelivery":         delivery.Redelivery,
		"parent_delivery_id": delivery.ParentDeliveryID,
		"delivered_at":       delivery.DeliveredAt,
		"created_at":         delivery.CreatedAt,
	})
	return dbErrors.MapDBError(err)
}

func (r *sqlxWebhookDeliveryRepository) GetByID(ctx context.Context, deliveryID, webhookID, orgID uuid.UUID) (*entity.WebhookDelivery, error) {
	query := `
		SELECT ` + webhookDeliverySelectColumns + `
		FROM webhook_deliveries
		WHERE id = ? AND webhook_id = ? AND organization_id = ?
	`
	query = r.DB.Rebind(query)

	delivery, err := r.scanDelivery(r.DB.QueryRowxContext(ctx, query, deliveryID, webhookID, orgID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return delivery, nil
}

func (r *sqlxWebhookDeliveryRepository) ListByWebhook(ctx context.Context, webhookID, orgID uuid.UUID, page, perPage int) ([]*entity.WebhookDelivery, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	countQuery := `
		SELECT COUNT(*)
		FROM webhook_deliveries
		WHERE webhook_id = ? AND organization_id = ?
	`
	countQuery = r.DB.Rebind(countQuery)

	var total int64
	if err := r.DB.GetContext(ctx, &total, countQuery, webhookID, orgID); err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}

	query := `
		SELECT ` + webhookDeliverySelectColumns + `
		FROM webhook_deliveries
		WHERE webhook_id = ? AND organization_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	query = r.DB.Rebind(query)

	rows, err := r.DB.QueryxContext(ctx, query, webhookID, orgID, perPage, offset)
	if err != nil {
		return nil, 0, dbErrors.MapDBError(err)
	}
	defer rows.Close()

	deliveries, err := r.scanDeliveryRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return deliveries, total, nil
}

func (r *sqlxWebhookDeliveryRepository) UpdateStatus(
	ctx context.Context,
	deliveryID uuid.UUID,
	status string,
	statusCode *int,
	responseHeaders map[string][]string,
	responseBody *string,
	durationMs *int,
	deliveredAt time.Time,
) error {
	responseHeadersJSON, err := marshalHeaderMap(responseHeaders)
	if err != nil {
		return err
	}

	truncatedBody := truncateResponseBody(responseBody)

	query := `
		UPDATE webhook_deliveries
		SET status = ?,
			status_code = ?,
			response_headers = ?,
			response_body = ?,
			duration_ms = ?,
			delivered_at = ?
		WHERE id = ?
	`
	query = r.DB.Rebind(query)

	result, err := r.DB.ExecContext(
		ctx,
		query,
		status,
		statusCode,
		responseHeadersJSON,
		truncatedBody,
		durationMs,
		deliveredAt,
		deliveryID,
	)
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

func truncateResponseBody(body *string) *string {
	if body == nil {
		return nil
	}
	if len(*body) <= maxWebhookDeliveryResponseBodyBytes {
		return body
	}
	truncated := (*body)[:maxWebhookDeliveryResponseBodyBytes]
	return &truncated
}

func marshalHeaderMap(headers map[string][]string) (string, error) {
	if headers == nil {
		return "{}", nil
	}
	data, err := json.Marshal(headers)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalHeaderMap(raw any) (map[string][]string, error) {
	switch value := raw.(type) {
	case nil:
		return map[string][]string{}, nil
	case []byte:
		return decodeHeaderMapJSON(value)
	case string:
		return decodeHeaderMapJSON([]byte(value))
	default:
		return map[string][]string{}, nil
	}
}

func decodeHeaderMapJSON(data []byte) (map[string][]string, error) {
	if len(data) == 0 {
		return map[string][]string{}, nil
	}
	var headers map[string][]string
	if err := json.Unmarshal(data, &headers); err != nil {
		return nil, err
	}
	if headers == nil {
		return map[string][]string{}, nil
	}
	return headers, nil
}

func (r *sqlxWebhookDeliveryRepository) scanDelivery(row *sqlx.Row) (*entity.WebhookDelivery, error) {
	var (
		delivery          entity.WebhookDelivery
		statusCode        sql.NullInt64
		requestHeadersRaw any
		responseHeadersRaw any
		responseBody      sql.NullString
		durationMs        sql.NullInt64
		parentDeliveryID  sql.NullString
		deliveredAt       nullTime
		createdAt         nullTime
	)

	err := row.Scan(
		&delivery.ID,
		&delivery.WebhookID,
		&delivery.OrganizationID,
		&delivery.Event,
		&delivery.Status,
		&statusCode,
		&requestHeadersRaw,
		&delivery.RequestBody,
		&responseHeadersRaw,
		&responseBody,
		&durationMs,
		&delivery.Attempt,
		&delivery.Redelivery,
		&parentDeliveryID,
		&deliveredAt,
		&createdAt,
	)
	if err != nil {
		return nil, err
	}
	delivery.CreatedAt = createdAt.Time

	if statusCode.Valid {
		code := int(statusCode.Int64)
		delivery.StatusCode = &code
	}
	requestHeaders, err := unmarshalHeaderMap(requestHeadersRaw)
	if err != nil {
		return nil, err
	}
	delivery.RequestHeaders = requestHeaders

	responseHeaders, err := unmarshalHeaderMap(responseHeadersRaw)
	if err != nil {
		return nil, err
	}
	delivery.ResponseHeaders = responseHeaders

	if responseBody.Valid {
		body := responseBody.String
		delivery.ResponseBody = &body
	}
	if durationMs.Valid {
		ms := int(durationMs.Int64)
		delivery.DurationMs = &ms
	}
	if parentDeliveryID.Valid {
		parsed, err := uuid.Parse(parentDeliveryID.String)
		if err != nil {
			return nil, err
		}
		delivery.ParentDeliveryID = &parsed
	}
	if deliveredAt.Valid {
		t := deliveredAt.Time
		delivery.DeliveredAt = &t
	}

	return &delivery, nil
}

func (r *sqlxWebhookDeliveryRepository) scanDeliveryRows(rows *sqlx.Rows) ([]*entity.WebhookDelivery, error) {
	var deliveries []*entity.WebhookDelivery
	for rows.Next() {
		var (
			delivery           entity.WebhookDelivery
			statusCode         sql.NullInt64
			requestHeadersRaw  any
			responseHeadersRaw any
			responseBody       sql.NullString
			durationMs         sql.NullInt64
			parentDeliveryID   sql.NullString
			deliveredAt        nullTime
			createdAt          nullTime
		)

		if err := rows.Scan(
			&delivery.ID,
			&delivery.WebhookID,
			&delivery.OrganizationID,
			&delivery.Event,
			&delivery.Status,
			&statusCode,
			&requestHeadersRaw,
			&delivery.RequestBody,
			&responseHeadersRaw,
			&responseBody,
			&durationMs,
			&delivery.Attempt,
			&delivery.Redelivery,
			&parentDeliveryID,
			&deliveredAt,
			&createdAt,
		); err != nil {
			return nil, dbErrors.MapDBError(err)
		}
		delivery.CreatedAt = createdAt.Time

		if statusCode.Valid {
			code := int(statusCode.Int64)
			delivery.StatusCode = &code
		}
		requestHeaders, err := unmarshalHeaderMap(requestHeadersRaw)
		if err != nil {
			return nil, err
		}
		delivery.RequestHeaders = requestHeaders

		responseHeaders, err := unmarshalHeaderMap(responseHeadersRaw)
		if err != nil {
			return nil, err
		}
		delivery.ResponseHeaders = responseHeaders

		if responseBody.Valid {
			body := responseBody.String
			delivery.ResponseBody = &body
		}
		if durationMs.Valid {
			ms := int(durationMs.Int64)
			delivery.DurationMs = &ms
		}
		if parentDeliveryID.Valid {
			parsed, err := uuid.Parse(parentDeliveryID.String)
			if err != nil {
				return nil, err
			}
			delivery.ParentDeliveryID = &parsed
		}
		if deliveredAt.Valid {
			t := deliveredAt.Time
			delivery.DeliveredAt = &t
		}

		deliveries = append(deliveries, &delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, dbErrors.MapDBError(err)
	}
	return deliveries, nil
}
