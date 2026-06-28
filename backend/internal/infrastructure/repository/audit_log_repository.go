package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

type sqlxAuditLogRepository struct {
	db *sqlx.DB
}

var (
	_ domainrepo.IAuditLogRepository = (*sqlxAuditLogRepository)(nil)
	_ domainrepo.AuditLogRepository  = (*sqlxAuditLogRepository)(nil)
)

func NewAuditLogRepository(db *sqlx.DB) domainrepo.AuditLogRepository {
	return &sqlxAuditLogRepository{db: db}
}

type AuditLogRow struct {
	ID             string    `db:"id"`
	OrganizationID string    `db:"organization_id"`
	ActorID        string    `db:"actor_id"`
	ActorLogin     string    `db:"actor_login"`
	Action         string    `db:"action"`
	TargetType     string    `db:"target_type"`
	TargetID       string    `db:"target_id"`
	IPAddress      string    `db:"ip_address"`
	Metadata       string    `db:"metadata"`
	CreatedAt      time.Time `db:"created_at"`
}

func AuditLogRowToEntity(row AuditLogRow) (*entity.AuditLog, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, err
	}
	orgID, err := uuid.Parse(row.OrganizationID)
	if err != nil {
		return nil, err
	}
	actorID, err := uuid.Parse(row.ActorID)
	if err != nil {
		return nil, err
	}

	var metadata map[string]any
	if row.Metadata != "" {
		if err := json.Unmarshal([]byte(row.Metadata), &metadata); err != nil {
			return nil, err
		}
	}

	return &entity.AuditLog{
		ID:             id,
		OrganizationID: orgID,
		ActorID:        actorID,
		ActorLogin:     row.ActorLogin,
		Action:         row.Action,
		TargetType:     row.TargetType,
		TargetID:       row.TargetID,
		Metadata:       metadata,
		CreatedAt:      row.CreatedAt,
	}, nil
}

func (r *sqlxAuditLogRepository) Create(ctx context.Context, log *entity.AuditLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	now := time.Now().UTC()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}

	metaJSON := []byte("{}")
	if log.Metadata != nil {
		encoded, err := json.Marshal(log.Metadata)
		if err != nil {
			return err
		}
		metaJSON = encoded
	}

	const query = `
		INSERT INTO audit_logs (id, organization_id, actor_id, actor_login, action, target_type, target_id, metadata, created_at)
		VALUES (:id, :organization_id, :actor_id, :actor_login, :action, :target_type, :target_id, :metadata, :created_at)
	`

	_, err := r.db.NamedExecContext(ctx, query, map[string]any{
		"id":              log.ID,
		"organization_id": log.OrganizationID,
		"actor_id":        log.ActorID,
		"actor_login":     log.ActorLogin,
		"action":          log.Action,
		"target_type":     log.TargetType,
		"target_id":       log.TargetID,
		"metadata":        string(metaJSON),
		"created_at":      log.CreatedAt,
	})
	return err
}

func (r *sqlxAuditLogRepository) InsertAuditLog(
	ctx context.Context,
	orgID, actorID uuid.UUID,
	action, targetType string,
	targetID uuid.UUID,
	metadata json.RawMessage,
) error {
	var meta map[string]any
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &meta); err != nil {
			return err
		}
	}

	return r.Create(ctx, &entity.AuditLog{
		OrganizationID: orgID,
		ActorID:        actorID,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID.String(),
		Metadata:       meta,
	})
}

func (r *sqlxAuditLogRepository) List(ctx context.Context, orgID uuid.UUID, action string, page, perPage int) ([]*entity.AuditLog, int, error) {
	baseQuery := `
		SELECT id, organization_id, actor_id, actor_login, action, target_type, target_id, metadata, created_at
		FROM audit_logs
		WHERE organization_id = :org_id
	`
	args := map[string]any{
		"org_id": orgID,
	}

	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE organization_id = :org_id`
	countArgs := map[string]any{
		"org_id": orgID,
	}

	if action != "" {
		baseQuery += ` AND action = :action`
		args["action"] = action
		countQuery += ` AND action = :action`
		countArgs["action"] = action
	}

	offset := (page - 1) * perPage
	listQuery := baseQuery + ` ORDER BY created_at DESC LIMIT :limit OFFSET :offset`
	args["limit"] = perPage
	args["offset"] = offset

	var total int
	countRows, err := r.db.NamedQueryContext(ctx, countQuery, countArgs)
	if err != nil {
		return nil, 0, err
	}
	defer countRows.Close()
	if countRows.Next() {
		if err := countRows.Scan(&total); err != nil {
			return nil, 0, err
		}
	}
	if err := countRows.Err(); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.NamedQueryContext(ctx, listQuery, args)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*entity.AuditLog
	for rows.Next() {
		var row AuditLogRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, err
		}
		log, err := AuditLogRowToEntity(row)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *sqlxAuditLogRepository) ListByOrg(ctx context.Context, opts domainrepo.AuditLogListOpts) ([]*entity.AuditLog, int64, error) {
	page := opts.Page
	if page < 1 {
		page = 1
	}
	perPage := opts.PerPage
	if perPage < 1 {
		perPage = 30
	}
	offset := (page - 1) * perPage

	where := "WHERE organization_id = ?"
	args := []any{opts.OrgID}

	if opts.Action != "" {
		where += " AND action = ?"
		args = append(args, opts.Action)
	}
	if opts.ActorID != nil {
		where += " AND actor_id = ?"
		args = append(args, *opts.ActorID)
	}
	if opts.Since != nil {
		where += " AND created_at >= ?"
		args = append(args, *opts.Since)
	}
	if opts.Until != nil {
		where += " AND created_at <= ?"
		args = append(args, *opts.Until)
	}

	countQuery := "SELECT COUNT(*) FROM audit_logs " + where
	countQuery = r.db.Rebind(countQuery)

	var total int64
	if err := r.db.QueryRowxContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	selectColumns := "id, organization_id, actor_id, actor_login, action, target_type, target_id, metadata, ip_address, created_at"
	listQuery := "SELECT " + selectColumns + " FROM audit_logs " + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	listQuery = r.db.Rebind(listQuery)
	listArgs := append(append([]any{}, args...), perPage, offset)

	rows, err := r.db.QueryxContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*entity.AuditLog
	for rows.Next() {
		var row AuditLogRow
		if err := rows.StructScan(&row); err != nil {
			return nil, 0, err
		}
		log, err := AuditLogRowToEntity(row)
		if err != nil {
			return nil, 0, err
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
