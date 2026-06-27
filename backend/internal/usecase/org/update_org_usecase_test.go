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
	testOrgID    = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	testOwnerID  = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testMemberID = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

type updateMockOrgRepo struct {
	byID    map[uuid.UUID]*entity.Organization
	updated *entity.Organization
}

func (m *updateMockOrgRepo) Create(context.Context, *entity.Organization) error { return nil }

func (m *updateMockOrgRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.Organization, error) {
	if o, ok := m.byID[id]; ok {
		return o, nil
	}
	return nil, domain.ErrNotFound
}

func (m *updateMockOrgRepo) GetByLogin(context.Context, string) (*entity.Organization, error) {
	return nil, domain.ErrNotFound
}

func (m *updateMockOrgRepo) List(context.Context, int, int) ([]*entity.Organization, error) {
	return nil, nil
}

func (m *updateMockOrgRepo) Update(_ context.Context, org *entity.Organization) error {
	m.updated = org
	return nil
}

func (m *updateMockOrgRepo) Delete(context.Context, uuid.UUID) error { return nil }

type updateMockMembershipRepo struct {
	roles map[string]string
}

func (m *updateMockMembershipRepo) roleKey(orgID, userID uuid.UUID) string {
	return orgID.String() + ":" + userID.String()
}

func (m *updateMockMembershipRepo) Add(context.Context, *entity.Membership) error { return nil }

func (m *updateMockMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", domain.ErrNotFound
	}
	role, ok := m.roles[m.roleKey(orgID, userID)]
	if !ok {
		return "", domain.ErrNotFound
	}
	return role, nil
}

func (m *updateMockMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *updateMockMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (m *updateMockMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func TestUpdateOrgMemberForbidden(t *testing.T) {
	orgs := &updateMockOrgRepo{
		byID: map[uuid.UUID]*entity.Organization{
			testOrgID: {ID: testOrgID, Login: "acme", Name: "Acme"},
		},
	}
	memberships := &updateMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testMemberID): entity.RoleMember,
		},
	}
	uc := org.NewUpdateOrgUsecase(orgs, memberships)

	_, err := uc.Execute(context.Background(), org.UpdateOrgInput{
		OrgID:       testOrgID,
		CallerID:    testMemberID,
		Name:        "New Name",
		Description: "New Description",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if orgs.updated != nil {
		t.Fatal("expected organization not to be updated")
	}
}

func TestUpdateOrgOwnerSuccess(t *testing.T) {
	orgs := &updateMockOrgRepo{
		byID: map[uuid.UUID]*entity.Organization{
			testOrgID: {ID: testOrgID, Login: "acme", Name: "Acme", Description: "Old"},
		},
	}
	memberships := &updateMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testOwnerID): entity.RoleOwner,
		},
	}
	uc := org.NewUpdateOrgUsecase(orgs, memberships)

	updated, err := uc.Execute(context.Background(), org.UpdateOrgInput{
		OrgID:       testOrgID,
		CallerID:    testOwnerID,
		Name:        "Acme Corp",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "Acme Corp" {
		t.Fatalf("expected name Acme Corp, got %s", updated.Name)
	}
	if updated.Description != "Updated description" {
		t.Fatalf("expected updated description, got %s", updated.Description)
	}
	if orgs.updated == nil {
		t.Fatal("expected organization to be updated")
	}
}

func membershipsRoleKey(orgID, userID uuid.UUID) string {
	return orgID.String() + ":" + userID.String()
}
