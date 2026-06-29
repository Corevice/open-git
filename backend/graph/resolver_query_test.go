package graph

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/open-git/backend/graph/globalid"
	"github.com/open-git/backend/graph/model"
	"github.com/open-git/backend/graph/relay"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/domain/repository"
	appmiddleware "github.com/open-git/backend/internal/middleware"
	infrarepo "github.com/open-git/backend/internal/repository"
	issueusecase "github.com/open-git/backend/internal/usecase/issue"
	repoUC "github.com/open-git/backend/internal/usecase/repository"
	userUC "github.com/open-git/backend/internal/usecase/user"
)

var (
	queryTestOrgID  = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	queryTestRepoID = uuid.MustParse("00000000-0000-0000-0000-000000000020")
	queryTestUserID = uuid.MustParse("00000000-0000-0000-0000-000000000030")
)

type queryMockDomainUserRepo struct {
	users map[int64]*domain.User
}

func (m *queryMockDomainUserRepo) Create(context.Context, *domain.User) error { return nil }

func (m *queryMockDomainUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	if m.users == nil {
		return nil, nil
	}
	return m.users[id], nil
}

func (m *queryMockDomainUserRepo) GetByLogin(context.Context, string) (*domain.User, error) {
	return nil, nil
}

func (m *queryMockDomainUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, nil
}

type queryMockRepositoryRepo struct {
	repo *entity.Repository
}

func (m *queryMockRepositoryRepo) Create(context.Context, *entity.Repository) error { return nil }

func (m *queryMockRepositoryRepo) GetByOwnerAndName(context.Context, uuid.UUID, string) (*entity.Repository, error) {
	return nil, nil
}

func (m *queryMockRepositoryRepo) GetByOwnerLoginAndName(context.Context, string, string) (*entity.Repository, error) {
	return m.repo, nil
}

func (m *queryMockRepositoryRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *queryMockRepositoryRepo) CountByOrg(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (m *queryMockRepositoryRepo) ListByOwner(context.Context, uuid.UUID, int, int) ([]*entity.Repository, error) {
	return nil, nil
}

func (m *queryMockRepositoryRepo) CountByOwner(context.Context, uuid.UUID) (int, error) { return 0, nil }

func (m *queryMockRepositoryRepo) UpdateVisibility(context.Context, uuid.UUID, string) error { return nil }

func (m *queryMockRepositoryRepo) UpdateName(context.Context, uuid.UUID, string) error { return nil }

func (m *queryMockRepositoryRepo) UpdateDefaultBranch(context.Context, uuid.UUID, string) error {
	return nil
}

func (m *queryMockRepositoryRepo) Delete(context.Context, uuid.UUID) error { return nil }

type queryMockMembershipRepo struct {
	hasAccess bool
}

func (m *queryMockMembershipRepo) HasReadAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return m.hasAccess, nil
}

func (m *queryMockMembershipRepo) HasWriteAccess(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return m.hasAccess, nil
}

type queryMockIssueRepo struct {
	issues     []*entity.Issue
	lastFilter repository.ListIssuesFilter
}

func (m *queryMockIssueRepo) Create(context.Context, *entity.Issue) error { return nil }

func (m *queryMockIssueRepo) GetByNumber(context.Context, uuid.UUID, int) (*entity.Issue, error) {
	return nil, nil
}

func (m *queryMockIssueRepo) GetByID(context.Context, uuid.UUID) (*entity.Issue, error) {
	return nil, nil
}

func (m *queryMockIssueRepo) ListByRepo(_ context.Context, filter repository.ListIssuesFilter) ([]*entity.Issue, int, error) {
	m.lastFilter = filter
	return m.issues, len(m.issues), nil
}

func (m *queryMockIssueRepo) Update(context.Context, *entity.Issue) error { return nil }

func (m *queryMockIssueRepo) Delete(context.Context, uuid.UUID) error { return nil }

func (m *queryMockIssueRepo) Count(context.Context, repository.ListIssuesFilter) (int, error) {
	return len(m.issues), nil
}

func (m *queryMockIssueRepo) NextNumber(context.Context, uuid.UUID) (int, error) { return 0, nil }

func newQueryTestResolver(userRepo infrarepo.IUserRepository, repoRepo infrarepo.IRepositoryRepository, membershipRepo infrarepo.IMembershipRepository, issueRepo repository.IIssueRepository) *Resolver {
	resolver := &Resolver{
		GetCurrentUserUC: userUC.NewGetCurrentUserUsecase(userRepo),
		GetRepositoryUC:  repoUC.NewGetRepositoryUsecase(repoRepo, userRepo, membershipRepo),
		ListIssuesUC:     issueusecase.NewListIssuesUsecase(issueRepo),
		GetIssueUC:       issueusecase.NewGetIssueUsecase(issueRepo),
	}
	return resolver
}

func TestQueryViewerReturnsCorrectUser(t *testing.T) {
	userID := int64(42)
	domainUser := &domain.User{
		ID:        userID,
		Login:     "octocat",
		CreatedAt: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	resolver := newQueryTestResolver(&queryMockDomainUserRepo{
		users: map[int64]*domain.User{userID: domainUser},
	}, &queryMockRepositoryRepo{}, &queryMockMembershipRepo{}, &queryMockIssueRepo{})

	viewerID := appmiddleware.Int64ToUUID(userID)
	ctx := WithViewer(context.Background(), &entity.User{ID: viewerID, Login: "octocat"})

	user, err := (&queryResolver{resolver}).Viewer(ctx)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.Equal(t, "octocat", user.Login)
	require.Equal(t, globalid.Encode(globalid.TypeUser, viewerID), user.ID)
}

func TestQueryRepositoryNotFound(t *testing.T) {
	resolver := newQueryTestResolver(
		&queryMockDomainUserRepo{},
		&queryMockRepositoryRepo{repo: nil},
		&queryMockMembershipRepo{},
		&queryMockIssueRepo{},
	)
	ctx := WithViewer(context.Background(), &entity.User{ID: queryTestUserID, Login: "octocat"})

	repo, err := (&queryResolver{resolver}).Repository(ctx, "missing", "repo")
	require.Error(t, err)
	require.Nil(t, repo)
	require.ErrorIs(t, err, repoUC.ErrNotFound)
}

func TestRepositoryIssuesConnectionRespectsFirstAfter(t *testing.T) {
	createdAt := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	issues := []*entity.Issue{
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"), RepositoryID: queryTestRepoID, OrganizationID: queryTestOrgID, Number: 1, Title: "One", State: "open", CreatedAt: createdAt},
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"), RepositoryID: queryTestRepoID, OrganizationID: queryTestOrgID, Number: 2, Title: "Two", State: "open", CreatedAt: createdAt.Add(time.Hour)},
		{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), RepositoryID: queryTestRepoID, OrganizationID: queryTestOrgID, Number: 3, Title: "Three", State: "open", CreatedAt: createdAt.Add(2 * time.Hour)},
	}
	issueRepo := &queryMockIssueRepo{issues: issues}
	resolver := newQueryTestResolver(&queryMockDomainUserRepo{}, &queryMockRepositoryRepo{}, &queryMockMembershipRepo{}, issueRepo)

	repoGraph := mapEntityRepository(&entity.Repository{
		ID:             queryTestRepoID,
		OrganizationID: queryTestOrgID,
		OwnerLogin:     "acme",
		Name:           "demo",
	})

	first := 1
	after := relay.EncodeCursor(issues[0].ID, issues[0].CreatedAt)
	conn, err := (&repositoryResolver{resolver}).Issues(context.Background(), repoGraph, &first, &after, nil, nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Len(t, conn.Edges, 1)
	require.Equal(t, 2, conn.Edges[0].Node.Number)
	require.True(t, conn.PageInfo.HasNextPage)
}

func TestQueryNodeMalformedIDReturnsNil(t *testing.T) {
	resolver := newQueryTestResolver(&queryMockDomainUserRepo{}, &queryMockRepositoryRepo{}, &queryMockMembershipRepo{}, &queryMockIssueRepo{})
	node, err := (&queryResolver{resolver}).Node(context.Background(), "%%%invalid-base64%%%")
	require.NoError(t, err)
	require.Nil(t, node)
}

func TestRepositoryIssuesListWithOpenStateFilter(t *testing.T) {
	issueRepo := &queryMockIssueRepo{
		issues: []*entity.Issue{
			{ID: uuid.New(), RepositoryID: queryTestRepoID, OrganizationID: queryTestOrgID, Number: 1, Title: "Bug", State: "open", CreatedAt: time.Now().UTC()},
		},
	}
	resolver := newQueryTestResolver(&queryMockDomainUserRepo{}, &queryMockRepositoryRepo{}, &queryMockMembershipRepo{}, issueRepo)
	repoGraph := mapEntityRepository(&entity.Repository{
		ID:             queryTestRepoID,
		OrganizationID: queryTestOrgID,
		OwnerLogin:     "acme",
		Name:           "demo",
	})

	first := 10
	_, err := (&repositoryResolver{resolver}).Issues(context.Background(), repoGraph, &first, nil, nil, nil, []model.IssueState{model.IssueStateOpen}, nil)
	require.NoError(t, err)
	require.Equal(t, "open", issueRepo.lastFilter.State)
}
