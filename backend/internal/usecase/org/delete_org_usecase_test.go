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

type deleteMockOrgRepo struct {
	deletedID uuid.UUID
}

func (m *deleteMockOrgRepo) Create(context.Context, *entity.Organization) error { return nil }

func (m *deleteMockOrgRepo) GetByID(context.Context, uuid.UUID) (*entity.Organization, error) {
	return nil, domain.ErrNotFound
}

func (m *deleteMockOrgRepo) GetByLogin(context.Context, string) (*entity.Organization, error) {
	return nil, domain.ErrNotFound
}

func (m *deleteMockOrgRepo) List(context.Context, int, int) ([]*entity.Organization, error) {
	return nil, nil
}

func (m *deleteMockOrgRepo) Update(context.Context, *entity.Organization) error { return nil }

func (m *deleteMockOrgRepo) Delete(_ context.Context, id uuid.UUID) error {
	m.deletedID = id
	return nil
}

type deleteMockMembershipRepo struct {
	roles map[string]string
}

func (m *deleteMockMembershipRepo) Add(context.Context, *entity.Membership) error { return nil }

func (m *deleteMockMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", domain.ErrNotFound
	}
	role, ok := m.roles[membershipsRoleKey(orgID, userID)]
	if !ok {
		return "", domain.ErrNotFound
	}
	return role, nil
}

func (m *deleteMockMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *deleteMockMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (m *deleteMockMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type deleteMockAuditLogRepo struct {
	callOrder []string
}

func (m *deleteMockAuditLogRepo) Record(_ context.Context, orgID, actorID uuid.UUID, action, targetType string, targetID uuid.UUID, _ map[string]any) error {
	m.callOrder = append(m.callOrder, "audit")
	return nil
}

func TestDeleteOrgNonOwnerForbidden(t *testing.T) {
	orgs := &deleteMockOrgRepo{}
	memberships := &deleteMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testMemberID): entity.RoleMember,
		},
	}
	auditLog := &deleteMockAuditLogRepo{}
	uc := org.NewDeleteOrgUsecase(orgs, memberships, auditLog)

	err := uc.Execute(context.Background(), org.DeleteOrgInput{
		OrgID:    testOrgID,
		CallerID: testMemberID,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if orgs.deletedID != uuid.Nil {
		t.Fatal("expected organization not to be deleted")
	}
	if len(auditLog.callOrder) != 0 {
		t.Fatal("expected audit log not to be recorded")
	}
}

func TestDeleteOrgOwnerSuccess(t *testing.T) {
	orgs := &deleteMockOrgRepo{}
	memberships := &deleteMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testOwnerID): entity.RoleOwner,
		},
	}
	auditLog := &deleteMockAuditLogRepo{}
	uc := org.NewDeleteOrgUsecase(orgs, memberships, auditLog)

	err := uc.Execute(context.Background(), org.DeleteOrgInput{
		OrgID:    testOrgID,
		CallerID: testOwnerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orgs.deletedID != testOrgID {
		t.Fatalf("expected org %s deleted, got %s", testOrgID, orgs.deletedID)
	}
	if len(auditLog.callOrder) != 1 || auditLog.callOrder[0] != "audit" {
		t.Fatalf("expected audit recorded before delete, got %v", auditLog.callOrder)
	}
}
