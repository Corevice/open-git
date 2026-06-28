package importjob_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	importjob "github.com/open-git/backend/internal/usecase/import"
)

var (
	testOrgID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testCallerID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testMemberID = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	testOrgLogin = "test-org"
)

type mockImportJobRepo struct {
	jobs       map[uuid.UUID]*entity.ImportJob
	created    []*entity.ImportJob
	createErr  error
	enqueuedID uuid.UUID
}

func (m *mockImportJobRepo) Create(_ context.Context, job *entity.ImportJob) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.jobs == nil {
		m.jobs = map[uuid.UUID]*entity.ImportJob{}
	}
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	m.jobs[job.ID] = job
	m.created = append(m.created, job)
	return nil
}

func (m *mockImportJobRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.ImportJob, error) {
	if job, ok := m.jobs[id]; ok {
		return job, nil
	}
	return nil, nil
}

func (m *mockImportJobRepo) GetByIDAndOrg(_ context.Context, id, orgID uuid.UUID) (*entity.ImportJob, error) {
	job, ok := m.jobs[id]
	if !ok || job.OrganizationID != orgID {
		return nil, fmt.Errorf("import job not found: %w", sql.ErrNoRows)
	}
	return job, nil
}

func (m *mockImportJobRepo) ListByOrg(_ context.Context, orgID uuid.UUID, page, perPage int) ([]*entity.ImportJob, int, error) {
	var jobs []*entity.ImportJob
	for _, job := range m.jobs {
		if job.OrganizationID == orgID {
			jobs = append(jobs, job)
		}
	}
	return jobs, len(jobs), nil
}

func (m *mockImportJobRepo) UpdateStatus(_ context.Context, id uuid.UUID, status entity.ImportJobStatus) error {
	job, ok := m.jobs[id]
	if !ok {
		return errors.New("not found")
	}
	job.Status = status
	return nil
}

func (m *mockImportJobRepo) UpdatePhase(_ context.Context, id uuid.UUID, phase entity.ImportJobPhase) error {
	job, ok := m.jobs[id]
	if !ok {
		return errors.New("not found")
	}
	job.Phase = phase
	return nil
}

func (m *mockImportJobRepo) UpdateProgress(context.Context, uuid.UUID, entity.ImportProgress) error {
	return nil
}

func (m *mockImportJobRepo) SetError(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *mockImportJobRepo) SetTargetRepository(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type mockMembershipRepo struct {
	roles map[string]string
}

func membershipKey(orgID, userID uuid.UUID) string {
	return fmt.Sprintf("%s:%s", orgID, userID)
}

func (m *mockMembershipRepo) Add(context.Context, *entity.Membership) error { return nil }

func (m *mockMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", domain.ErrNotFound
	}
	role, ok := m.roles[membershipKey(orgID, userID)]
	if !ok {
		return "", domain.ErrNotFound
	}
	return role, nil
}

func (m *mockMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *mockMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error { return nil }

func (m *mockMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type mockRepositoryRepo struct {
	byLoginAndName map[string]*entity.Repository
}

func repoLoginKey(ownerLogin, name string) string {
	return fmt.Sprintf("%s:%s", ownerLogin, name)
}

func (m *mockRepositoryRepo) Create(context.Context, *entity.Repository) error { return nil }

func (m *mockRepositoryRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, errors.New("not found")
}

func (m *mockRepositoryRepo) GetByOwnerLoginAndName(_ context.Context, ownerLogin, name string) (*entity.Repository, error) {
	if m.byLoginAndName == nil {
		return nil, errors.New("not found")
	}
	if repo, ok := m.byLoginAndName[repoLoginKey(ownerLogin, name)]; ok {
		return repo, nil
	}
	return nil, errors.New("not found")
}

func (m *mockRepositoryRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryRepo) CountByOrg(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockRepositoryRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *mockRepositoryRepo) CountByOwner(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *mockRepositoryRepo) UpdateName(context.Context, uuid.UUID, string) error { return nil }

func (m *mockRepositoryRepo) Delete(context.Context, uuid.UUID) error { return nil }

type mockOrgRepo struct {
	org *entity.Organization
}

func (m *mockOrgRepo) Create(context.Context, *entity.Organization) error { return nil }

func (m *mockOrgRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Organization, error) {
	if m.org != nil && m.org.ID == id {
		return m.org, nil
	}
	return nil, nil
}

func (m *mockOrgRepo) GetByLogin(context.Context, string) (*entity.Organization, error) {
	return nil, nil
}

func (m *mockOrgRepo) List(context.Context, int, int) ([]*entity.Organization, error) {
	return nil, nil
}

func (m *mockOrgRepo) Update(context.Context, *entity.Organization) error { return nil }

func (m *mockOrgRepo) Delete(context.Context, uuid.UUID) error { return nil }

type mockSecretStorer struct {
	stored map[string]string
}

func (m *mockSecretStorer) StoreSecret(_ context.Context, ref, value string) error {
	if m.stored == nil {
		m.stored = map[string]string{}
	}
	m.stored[ref] = value
	return nil
}

type mockEnqueuer struct {
	called bool
	jobID  uuid.UUID
	orgID  uuid.UUID
	err    error
}

func (m *mockEnqueuer) EnqueueGitHubImport(_ context.Context, jobID, organizationID uuid.UUID) error {
	m.called = true
	m.jobID = jobID
	m.orgID = organizationID
	return m.err
}

func newCreateUsecase(
	importJobs domainrepo.IImportJobRepository,
	memberships domainrepo.IMembershipRepository,
	repositories *mockRepositoryRepo,
	orgs domainrepo.IOrganizationRepository,
	enqueuer importjob.GitHubImportEnqueuer,
) *importjob.CreateImportJobUsecase {
	return importjob.NewCreateImportJobUsecaseWithDeps(
		importJobs,
		memberships,
		repositories,
		orgs,
		&mockSecretStorer{},
		enqueuer,
	)
}

func TestCreateImportJobHappyPath(t *testing.T) {
	importJobs := &mockImportJobRepo{jobs: map[uuid.UUID]*entity.ImportJob{}}
	memberships := &mockMembershipRepo{
		roles: map[string]string{
			membershipKey(testOrgID, testCallerID): entity.RoleAdmin,
		},
	}
	repositories := &mockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	orgs := &mockOrgRepo{
		org: &entity.Organization{ID: testOrgID, Login: testOrgLogin},
	}
	enqueuer := &mockEnqueuer{}
	uc := newCreateUsecase(importJobs, memberships, repositories, orgs, enqueuer)

	job, err := uc.Execute(context.Background(), importjob.CreateImportJobInput{
		OrganizationID: testOrgID,
		CallerID:       testCallerID,
		SourceURL:      "https://github.com/acme/source-repo",
		TargetName:     "source-repo",
		Include:        []string{"code", "issues"},
		GitHubToken:    "ghp_test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if job.Status != entity.ImportJobStatusQueued {
		t.Fatalf("expected queued status, got %s", job.Status)
	}
	if job.Phase != entity.ImportJobPhaseClone {
		t.Fatalf("expected clone phase, got %s", job.Phase)
	}
	if len(importJobs.created) != 1 {
		t.Fatal("expected job to be created")
	}
	if !enqueuer.called {
		t.Fatal("expected import task to be enqueued")
	}
	if enqueuer.jobID != job.ID {
		t.Fatalf("expected enqueued job id %s, got %s", job.ID, enqueuer.jobID)
	}
}

func TestCreateImportJobEmptyInclude(t *testing.T) {
	importJobs := &mockImportJobRepo{jobs: map[uuid.UUID]*entity.ImportJob{}}
	memberships := &mockMembershipRepo{
		roles: map[string]string{
			membershipKey(testOrgID, testCallerID): entity.RoleAdmin,
		},
	}
	repositories := &mockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	orgs := &mockOrgRepo{
		org: &entity.Organization{ID: testOrgID, Login: testOrgLogin},
	}
	enqueuer := &mockEnqueuer{}
	uc := newCreateUsecase(importJobs, memberships, repositories, orgs, enqueuer)

	_, err := uc.Execute(context.Background(), importjob.CreateImportJobInput{
		OrganizationID: testOrgID,
		CallerID:       testCallerID,
		SourceURL:      "https://github.com/acme/source-repo",
		TargetName:     "source-repo",
		Include:        nil,
	})
	if err == nil {
		t.Fatal("expected validation error for empty include")
	}
	if len(importJobs.created) != 0 {
		t.Fatal("expected no job to be created")
	}
	if enqueuer.called {
		t.Fatal("expected import task not to be enqueued")
	}
}

func TestCreateImportJobTargetNameConflict(t *testing.T) {
	importJobs := &mockImportJobRepo{jobs: map[uuid.UUID]*entity.ImportJob{}}
	memberships := &mockMembershipRepo{
		roles: map[string]string{
			membershipKey(testOrgID, testCallerID): entity.RoleAdmin,
		},
	}
	repositories := &mockRepositoryRepo{
		byLoginAndName: map[string]*entity.Repository{
			repoLoginKey(testOrgLogin, "taken"): {Name: "taken"},
		},
	}
	orgs := &mockOrgRepo{
		org: &entity.Organization{ID: testOrgID, Login: testOrgLogin},
	}
	enqueuer := &mockEnqueuer{}
	uc := newCreateUsecase(importJobs, memberships, repositories, orgs, enqueuer)

	_, err := uc.Execute(context.Background(), importjob.CreateImportJobInput{
		OrganizationID: testOrgID,
		CallerID:       testCallerID,
		SourceURL:      "https://github.com/acme/source-repo",
		TargetName:     "taken",
		Include:        []string{"code"},
	})
	if !errors.Is(err, importjob.ErrTargetNameConflict) {
		t.Fatalf("expected ErrTargetNameConflict, got %v", err)
	}
	if len(importJobs.created) != 0 {
		t.Fatal("expected no job to be created")
	}
}

func TestCreateImportJobNonAdminForbidden(t *testing.T) {
	importJobs := &mockImportJobRepo{jobs: map[uuid.UUID]*entity.ImportJob{}}
	memberships := &mockMembershipRepo{
		roles: map[string]string{
			membershipKey(testOrgID, testMemberID): entity.RoleMember,
		},
	}
	repositories := &mockRepositoryRepo{byLoginAndName: map[string]*entity.Repository{}}
	orgs := &mockOrgRepo{
		org: &entity.Organization{ID: testOrgID, Login: testOrgLogin},
	}
	enqueuer := &mockEnqueuer{}
	uc := newCreateUsecase(importJobs, memberships, repositories, orgs, enqueuer)

	_, err := uc.Execute(context.Background(), importjob.CreateImportJobInput{
		OrganizationID: testOrgID,
		CallerID:       testMemberID,
		SourceURL:      "https://github.com/acme/source-repo",
		TargetName:     "source-repo",
		Include:        []string{"code"},
	})
	if !errors.Is(err, importjob.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if len(importJobs.created) != 0 {
		t.Fatal("expected no job to be created")
	}
}
