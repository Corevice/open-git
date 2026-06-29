package graph

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/open-git/backend/graph/dataloader"
	"github.com/open-git/backend/graph/globalid"
	"github.com/open-git/backend/graph/model"
	"github.com/open-git/backend/internal/apperror"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
)

var (
	mutationTestOrgID      = uuid.MustParse("00000000-0000-0000-0000-000000000101")
	mutationTestRepoID     = uuid.MustParse("00000000-0000-0000-0000-000000000102")
	mutationTestUserID     = uuid.MustParse("00000000-0000-0000-0000-000000000103")
	mutationTestIssueID    = uuid.MustParse("00000000-0000-0000-0000-000000000104")
	mutationTestPRID       = uuid.MustParse("00000000-0000-0000-0000-000000000105")
	mutationOtherLabelID   = uuid.MustParse("00000000-0000-0000-0000-000000000107")
	mutationOtherRepoID    = uuid.MustParse("00000000-0000-0000-0000-000000000108")
)

type mutationAuditLogRepo struct{}

func (mutationAuditLogRepo) InsertAuditLog(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, json.RawMessage) error {
	return nil
}

type mutationTxManager struct{}

func (mutationTxManager) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mutationIssueRepo struct {
	issues map[uuid.UUID]*entity.Issue
}

func (m *mutationIssueRepo) Create(context.Context, *entity.Issue) error { return nil }

func (m *mutationIssueRepo) GetByNumber(context.Context, uuid.UUID, int) (*entity.Issue, error) {
	return nil, nil
}

func (m *mutationIssueRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Issue, error) {
	if m.issues == nil {
		return nil, nil
	}
	return m.issues[id], nil
}

func (m *mutationIssueRepo) ListByRepo(context.Context, repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	return nil, 0, nil
}

func (m *mutationIssueRepo) Update(context.Context, *entity.Issue) error { return nil }

func (m *mutationIssueRepo) Delete(context.Context, uuid.UUID) error { return nil }

func (m *mutationIssueRepo) Count(context.Context, repository.ListIssuesFilter) (int, error) {
	return 0, nil
}

func (m *mutationIssueRepo) NextNumber(context.Context, uuid.UUID) (int, error) {
	return 1, nil
}

type mutationPRRepo struct {
	prs map[uuid.UUID]*entity.PullRequest
}

func (m *mutationPRRepo) Create(context.Context, *entity.PullRequest) error { return nil }

func (m *mutationPRRepo) GetByNumber(context.Context, uuid.UUID, int) (*entity.PullRequest, error) {
	return nil, nil
}

func (m *mutationPRRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.PullRequest, error) {
	if m.prs == nil {
		return nil, nil
	}
	return m.prs[id], nil
}

func (m *mutationPRRepo) ListByRepo(context.Context, uuid.UUID, repository.ListPullRequestsFilter) ([]*entity.PullRequest, int, error) {
	return nil, 0, nil
}

func (m *mutationPRRepo) NextNumber(context.Context, uuid.UUID) (int, error) { return 1, nil }

func (m *mutationPRRepo) Update(context.Context, *entity.PullRequest) error { return nil }

func (m *mutationPRRepo) SetMerged(context.Context, uuid.UUID, time.Time, uuid.UUID, string) error {
	return nil
}

type mutationRepoRepo struct {
	repos map[uuid.UUID]*entity.Repository
}

func (m *mutationRepoRepo) Create(context.Context, *entity.Repository) error { return nil }

func (m *mutationRepoRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, nil
}

func (m *mutationRepoRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, int, error) {
	return nil, 0, nil
}

func (m *mutationRepoRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *mutationRepoRepo) Delete(context.Context, uuid.UUID) error { return nil }

func (m *mutationRepoRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Repository, error) {
	if m.repos == nil {
		return nil, nil
	}
	return m.repos[id], nil
}

type mutationLabelRepo struct {
	labels map[uuid.UUID]*entity.Label
}

func (m *mutationLabelRepo) Create(context.Context, *entity.Label) error { return nil }

func (m *mutationLabelRepo) GetByName(context.Context, uuid.UUID, string) (*entity.Label, error) {
	return nil, nil
}

func (m *mutationLabelRepo) ListByRepo(context.Context, uuid.UUID, int, int) ([]*entity.Label, int, error) {
	return nil, 0, nil
}

func (m *mutationLabelRepo) Update(context.Context, *entity.Label) error { return nil }

func (m *mutationLabelRepo) Delete(context.Context, uuid.UUID) error { return nil }

func (m *mutationLabelRepo) AddToIssue(context.Context, uuid.UUID, int, uuid.UUID) error { return nil }

func (m *mutationLabelRepo) RemoveFromIssue(context.Context, uuid.UUID, int, uuid.UUID) error {
	return nil
}

func (m *mutationLabelRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Label, error) {
	if m.labels == nil {
		return nil, nil
	}
	return m.labels[id], nil
}

type mutationUserRepo struct{}

func (mutationUserRepo) Create(context.Context, *entity.User) error { return nil }
func (mutationUserRepo) Update(context.Context, *entity.User) error { return nil }
func (mutationUserRepo) GetByLogin(context.Context, string) (*entity.User, error) {
	return nil, nil
}
func (mutationUserRepo) GetByEmail(context.Context, string) (*entity.User, error) {
	return nil, nil
}
func (mutationUserRepo) GetByID(context.Context, uuid.UUID) (*entity.User, error) {
	return nil, nil
}

type stubMilestoneRepo struct{}

func (stubMilestoneRepo) Create(context.Context, *entity.Milestone) error { return nil }
func (stubMilestoneRepo) GetByNumber(context.Context, uuid.UUID, int) (*entity.Milestone, error) {
	return nil, nil
}
func (stubMilestoneRepo) ListByRepo(context.Context, uuid.UUID, string, int, int) ([]*entity.Milestone, int, error) {
	return nil, 0, nil
}
func (stubMilestoneRepo) Update(context.Context, *entity.Milestone) error { return nil }
func (stubMilestoneRepo) Delete(context.Context, uuid.UUID) error         { return nil }
func (stubMilestoneRepo) NextNumber(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}
func (stubMilestoneRepo) IncrOpenCount(context.Context, uuid.UUID) error { return nil }
func (stubMilestoneRepo) DecrOpenCount(context.Context, uuid.UUID) error { return nil }

func mutationTestContext(userRepo *mutationUserRepo, labelRepo *mutationLabelRepo, repoRepo *mutationRepoRepo) context.Context {
	ctx := context.Background()
	ctx = WithViewer(ctx, &entity.User{ID: mutationTestUserID, Login: "octocat"})
	ctx = WithScopes(ctx, []string{ScopeRepo})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/graphql", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	mw := dataloader.Middleware(userRepo, labelRepo, &stubMilestoneRepo{}, repoRepo)
	_ = mw(func(c echo.Context) error { return nil })(c)
	if loaders := dataloader.FromEcho(c); loaders != nil {
		ctx = WithLoaders(ctx, loaders)
	}
	return ctx
}

func TestCreateIssueEmptyTitleReturnsUnprocessable(t *testing.T) {
	repoRepo := &mutationRepoRepo{
		repos: map[uuid.UUID]*entity.Repository{
			mutationTestRepoID: {
				ID:             mutationTestRepoID,
				OrganizationID: mutationTestOrgID,
			},
		},
	}
	resolver := &Resolver{
		CreateIssueUC: issueusecase.NewCreateIssueUsecase(
			&mutationIssueRepo{},
			mutationAuditLogRepo{},
			mutationTxManager{},
		),
	}
	ctx := mutationTestContext(&mutationUserRepo{}, &mutationLabelRepo{}, repoRepo)

	_, err := resolver.Mutation().CreateIssue(ctx, model.CreateIssueInput{
		RepositoryID: globalid.Encode(globalid.TypeRepository, mutationTestRepoID),
		Title:        "",
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, apperror.ErrValidation))
}

func TestCreateIssueSuccessEchoesClientMutationID(t *testing.T) {
	repoRepo := &mutationRepoRepo{
		repos: map[uuid.UUID]*entity.Repository{
			mutationTestRepoID: {
				ID:             mutationTestRepoID,
				OrganizationID: mutationTestOrgID,
				OwnerLogin:     "octocat",
				Name:           "hello-world",
			},
		},
	}
	resolver := &Resolver{
		CreateIssueUC: issueusecase.NewCreateIssueUsecase(
			&mutationIssueRepo{},
			mutationAuditLogRepo{},
			mutationTxManager{},
		),
	}
	ctx := mutationTestContext(&mutationUserRepo{}, &mutationLabelRepo{}, repoRepo)
	clientMutationID := "mutation-123"

	payload, err := resolver.Mutation().CreateIssue(ctx, model.CreateIssueInput{
		RepositoryID:     globalid.Encode(globalid.TypeRepository, mutationTestRepoID),
		Title:            "Bug report",
		ClientMutationID: &clientMutationID,
	})
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotNil(t, payload.Issue)
	require.NotNil(t, payload.ClientMutationID)
	require.Equal(t, clientMutationID, *payload.ClientMutationID)
	require.Equal(t, "Bug report", payload.Issue.Title)
}

func TestMergePullRequestConflictingReturnsUnprocessable(t *testing.T) {
	repoRepo := &mutationRepoRepo{
		repos: map[uuid.UUID]*entity.Repository{
			mutationTestRepoID: {
				ID:             mutationTestRepoID,
				OrganizationID: mutationTestOrgID,
				GitPath:        "/data/repo.git",
			},
		},
	}
	resolver := &Resolver{
		PullRequestRepo: &mutationPRRepo{
			prs: map[uuid.UUID]*entity.PullRequest{
				mutationTestPRID: {
					ID:             mutationTestPRID,
					OrganizationID: mutationTestOrgID,
					RepositoryID:   mutationTestRepoID,
					Number:         1,
					MergeableState: entity.MergeableStateDirty,
					State:          entity.PullRequestStateOpen,
				},
			},
		},
	}
	ctx := mutationTestContext(&mutationUserRepo{}, &mutationLabelRepo{}, repoRepo)

	_, err := resolver.Mutation().MergePullRequest(ctx, model.MergePullRequestInput{
		PullRequestID: globalid.Encode(globalid.TypePullRequest, mutationTestPRID),
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, apperror.ErrValidation))
}

func TestAddLabelsToLabelableWrongRepoReturnsNotFound(t *testing.T) {
	labelRepo := &mutationLabelRepo{
		labels: map[uuid.UUID]*entity.Label{
			mutationOtherLabelID: {
				ID:           mutationOtherLabelID,
				RepositoryID: mutationOtherRepoID,
				Name:         "bug",
				Color:        "ff0000",
			},
		},
	}
	resolver := &Resolver{
		IssueRepo: &mutationIssueRepo{
			issues: map[uuid.UUID]*entity.Issue{
				mutationTestIssueID: {
					ID:             mutationTestIssueID,
					OrganizationID: mutationTestOrgID,
					RepositoryID:   mutationTestRepoID,
					Number:         7,
					Title:          "Issue",
					State:          "open",
				},
			},
		},
		LabelRepo: labelRepo,
	}
	ctx := mutationTestContext(&mutationUserRepo{}, labelRepo, &mutationRepoRepo{})

	_, err := resolver.Mutation().AddLabelsToLabelable(ctx, model.AddLabelsToLabelableInput{
		LabelableID: globalid.Encode(globalid.TypeIssue, mutationTestIssueID),
		LabelIds:    []string{globalid.Encode(globalid.TypeLabel, mutationOtherLabelID)},
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, apperror.ErrNotFound))
}

func TestCloseIssueAlreadyClosedIsIdempotent(t *testing.T) {
	closedAt := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	issue := &entity.Issue{
		ID:             mutationTestIssueID,
		OrganizationID: mutationTestOrgID,
		RepositoryID:   mutationTestRepoID,
		Number:         3,
		Title:          "Closed issue",
		State:          "closed",
		ClosedAt:       &closedAt,
		CreatedAt:      closedAt,
		UpdatedAt:      closedAt,
	}
	resolver := &Resolver{
		IssueRepo: &mutationIssueRepo{
			issues: map[uuid.UUID]*entity.Issue{
				mutationTestIssueID: issue,
			},
		},
		UpdateIssueUC: issueusecase.NewUpdateIssueUsecase(
			&mutationIssueRepo{issues: map[uuid.UUID]*entity.Issue{mutationTestIssueID: issue}},
			&mutationLabelRepo{},
			&stubMilestoneRepo{},
			mutationAuditLogRepo{},
		),
	}
	ctx := mutationTestContext(&mutationUserRepo{}, &mutationLabelRepo{}, &mutationRepoRepo{})

	payload, err := resolver.Mutation().CloseIssue(ctx, model.CloseIssueInput{
		IssueID: globalid.Encode(globalid.TypeIssue, mutationTestIssueID),
	})
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotNil(t, payload.Issue)
	require.Equal(t, model.IssueStateClosed, payload.Issue.State)
}
