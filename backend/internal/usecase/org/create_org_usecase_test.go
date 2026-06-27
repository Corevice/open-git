package org_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/open-git/backend/internal/domain"
	"github.com/open-git/backend/internal/domain/entity"
	"github.com/open-git/backend/internal/usecase/org"
)

var (
	testCreatorID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

type mockOrgRepo struct {
	byLogin map[string]*entity.Organization
	created []*entity.Organization
}

func (m *mockOrgRepo) Create(_ context.Context, org *entity.Organization) error {
	if m.byLogin == nil {
		m.byLogin = map[string]*entity.Organization{}
	}
	org.ID = uuid.New()
	m.created = append(m.created, org)
	m.byLogin[org.Login] = org
	return nil
}

func (m *mockOrgRepo) GetByID(context.Context, uuid.UUID) (*entity.Organization, error) {
	return nil, domain.ErrNotFound
}

func (m *mockOrgRepo) GetByLogin(_ context.Context, login string) (*entity.Organization, error) {
	if o, ok := m.byLogin[login]; ok {
		return o, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockOrgRepo) List(context.Context, int, int) ([]*entity.Organization, error) {
	return nil, nil
}

func (m *mockOrgRepo) Update(context.Context, *entity.Organization) error {
	return nil
}

func (m *mockOrgRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

type mockMembershipRepo struct {
	added []*entity.Membership
}

func (m *mockMembershipRepo) Add(_ context.Context, membership *entity.Membership) error {
	m.added = append(m.added, membership)
	return nil
}

func (m *mockMembershipRepo) GetRole(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", domain.ErrNotFound
}

func (m *mockMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *mockMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (m *mockMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func TestCreateOrgHappyPath(t *testing.T) {
	orgs := &mockOrgRepo{byLogin: map[string]*entity.Organization{}}
	memberships := &mockMembershipRepo{}
	uc := org.NewCreateOrgUsecase(orgs, memberships)

	created, err := uc.Execute(context.Background(), org.CreateOrgInput{
		CreatorID:   testCreatorID,
		Login:       "my-org",
		Name:        "My Org",
		Description: "A test organization",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Login != "my-org" {
		t.Fatalf("expected login my-org, got %s", created.Login)
	}
	if created.Description != "A test organization" {
		t.Fatalf("expected description, got %s", created.Description)
	}
	if len(orgs.created) != 1 {
		t.Fatal("expected organization to be created")
	}
	if len(memberships.added) != 1 {
		t.Fatal("expected owner membership to be added")
	}
	membership := memberships.added[0]
	if membership.UserID != testCreatorID {
		t.Fatalf("expected creator %s, got %s", testCreatorID, membership.UserID)
	}
	if membership.OrganizationID != created.ID {
		t.Fatalf("expected org id %s, got %s", created.ID, membership.OrganizationID)
	}
	if membership.Role != entity.RoleOwner {
		t.Fatalf("expected owner role, got %s", membership.Role)
	}
}

func TestCreateOrgDuplicateLogin(t *testing.T) {
	orgs := &mockOrgRepo{
		byLogin: map[string]*entity.Organization{
			"existing": {Login: "existing"},
		},
	}
	memberships := &mockMembershipRepo{}
	uc := org.NewCreateOrgUsecase(orgs, memberships)

	_, err := uc.Execute(context.Background(), org.CreateOrgInput{
		CreatorID: testCreatorID,
		Login:     "existing",
		Name:      "Existing Org",
	})
	if !errors.Is(err, org.ErrDuplicateLogin) {
		t.Fatalf("expected ErrDuplicateLogin, got %v", err)
	}
	if len(memberships.added) != 0 {
		t.Fatal("expected no membership to be added")
	}
}

func TestCreateOrgReservedLogin(t *testing.T) {
	orgs := &mockOrgRepo{byLogin: map[string]*entity.Organization{}}
	memberships := &mockMembershipRepo{}
	uc := org.NewCreateOrgUsecase(orgs, memberships)

	_, err := uc.Execute(context.Background(), org.CreateOrgInput{
		CreatorID: testCreatorID,
		Login:     "admin",
		Name:      "Admin Org",
	})
	if !errors.Is(err, org.ErrReservedLogin) {
		t.Fatalf("expected ErrReservedLogin, got %v", err)
	}
	if len(orgs.created) != 0 {
		t.Fatal("expected no organization to be created")
	}
	if len(memberships.added) != 0 {
		t.Fatal("expected no membership to be added")
	}
}
