package importjob

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	repo "github.com/open-git/backend/internal/repository"
)

const typeGitHubImport = "import:github"

var (
	githubSourceURLRegex = regexp.MustCompile(`^https://github\.com/[^/]+/[^/]+(/.*)?$`)
	targetNameRegex      = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
)

type SecretStorer interface {
	StoreSecret(ctx context.Context, ref, value string) error
}

type noopSecretStorer struct{}

func (noopSecretStorer) StoreSecret(context.Context, string, string) error {
	return nil
}

type GitHubImportEnqueuer interface {
	EnqueueGitHubImport(ctx context.Context, jobID, organizationID uuid.UUID) error
}

type asynqGitHubImportEnqueuer struct {
	client *asynq.Client
}

func newAsynqGitHubImportEnqueuer(client *asynq.Client) GitHubImportEnqueuer {
	return &asynqGitHubImportEnqueuer{client: client}
}

type githubImportTaskPayload struct {
	ImportJobID    string `json:"import_job_id"`
	OrganizationID string `json:"organization_id"`
}

func (e *asynqGitHubImportEnqueuer) EnqueueGitHubImport(ctx context.Context, jobID, organizationID uuid.UUID) error {
	payload := githubImportTaskPayload{
		ImportJobID:    jobID.String(),
		OrganizationID: organizationID.String(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal import task payload: %w", err)
	}
	task := asynq.NewTask(typeGitHubImport, data)
	_, err = e.client.EnqueueContext(ctx, task)
	return err
}

type CreateImportJobInput struct {
	OrganizationID uuid.UUID
	CallerID       uuid.UUID
	SourceURL      string
	TargetName     string
	Include        []string
	GitHubToken    string
}

type CreateImportJobUsecase struct {
	importJobs   domainrepo.IImportJobRepository
	memberships  domainrepo.IMembershipRepository
	repositories repo.IRepositoryRepository
	orgs         domainrepo.IOrganizationRepository
	secrets      SecretStorer
	enqueuer     GitHubImportEnqueuer
}

func NewCreateImportJobUsecase(
	importJobs domainrepo.IImportJobRepository,
	memberships domainrepo.IMembershipRepository,
	repositories repo.IRepositoryRepository,
	orgs domainrepo.IOrganizationRepository,
	client *asynq.Client,
) *CreateImportJobUsecase {
	return NewCreateImportJobUsecaseWithDeps(
		importJobs,
		memberships,
		repositories,
		orgs,
		noopSecretStorer{},
		newAsynqGitHubImportEnqueuer(client),
	)
}

func NewCreateImportJobUsecaseWithDeps(
	importJobs domainrepo.IImportJobRepository,
	memberships domainrepo.IMembershipRepository,
	repositories repo.IRepositoryRepository,
	orgs domainrepo.IOrganizationRepository,
	secrets SecretStorer,
	enqueuer GitHubImportEnqueuer,
) *CreateImportJobUsecase {
	if secrets == nil {
		secrets = noopSecretStorer{}
	}
	return &CreateImportJobUsecase{
		importJobs:   importJobs,
		memberships:  memberships,
		repositories: repositories,
		orgs:         orgs,
		secrets:      secrets,
		enqueuer:     enqueuer,
	}
}

func (u *CreateImportJobUsecase) Execute(ctx context.Context, input CreateImportJobInput) (*entity.ImportJob, error) {
	if !githubSourceURLRegex.MatchString(input.SourceURL) {
		return nil, ErrInvalidSourceURL
	}
	if !targetNameRegex.MatchString(input.TargetName) {
		return nil, fmt.Errorf("invalid target name")
	}
	if len(input.Include) == 0 {
		return nil, fmt.Errorf("include must not be empty")
	}

	if err := u.checkCallerAdmin(ctx, input.OrganizationID, input.CallerID); err != nil {
		return nil, err
	}

	org, err := u.orgs.GetByID(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrNotFound
	}

	if _, err := u.repositories.GetByOwnerLoginAndName(ctx, org.Login, input.TargetName); err == nil {
		return nil, ErrTargetNameConflict
	} else if !isRepositoryNotFound(err) {
		return nil, err
	}

	jobID := uuid.New()
	now := time.Now().UTC()

	var tokenRef *string
	if input.GitHubToken != "" {
		secretRef := fmt.Sprintf("import/%s/github-token", jobID)
		if err := u.secrets.StoreSecret(ctx, secretRef, input.GitHubToken); err != nil {
			return nil, err
		}
		tokenRef = &secretRef
	}

	job := &entity.ImportJob{
		ID:             jobID,
		OrganizationID: input.OrganizationID,
		CreatedBy:      input.CallerID,
		SourceURL:      input.SourceURL,
		TargetName:     input.TargetName,
		Include:        append([]string(nil), input.Include...),
		Status:         entity.ImportJobStatusQueued,
		Phase:          entity.ImportJobPhaseClone,
		Progress:       entity.ImportProgress{},
		TokenSecretRef: tokenRef,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := u.importJobs.Create(ctx, job); err != nil {
		return nil, err
	}

	if err := u.enqueuer.EnqueueGitHubImport(ctx, job.ID, input.OrganizationID); err != nil {
		return nil, err
	}

	return job, nil
}

func (u *CreateImportJobUsecase) checkCallerAdmin(ctx context.Context, organizationID, callerID uuid.UUID) error {
	role, err := u.memberships.GetRole(ctx, organizationID, callerID)
	if errors.Is(err, domain.ErrNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if role != entity.RoleOwner && role != entity.RoleAdmin {
		return ErrForbidden
	}
	return nil
}

func isRepositoryNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || (err != nil && err.Error() == "not found")
}
