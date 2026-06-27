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

type removeMockMembershipRepo struct {
	roles      map[string]string
	members    []*entity.Membership
	removedOrg uuid.UUID
	removedUser uuid.UUID
}

func (m *removeMockMembershipRepo) Add(context.Context, *entity.Membership) error { return nil }

func (m *removeMockMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", domain.ErrNotFound
	}
	role, ok := m.roles[membershipsRoleKey(orgID, userID)]
	if !ok {
		return "", domain.ErrNotFound
	}
	return role, nil
}

func (m *removeMockMembershipRepo) ListByOrg(_ context.Context, orgID uuid.UUID, _, _ int) ([]*entity.Membership, error) {
	if m.members != nil {
		return m.members, nil
	}
	var result []*entity.Membership
	for key, role := range m.roles {
		parsedOrgID, parsedUserID, err := parseMembershipRoleKey(key)
		if err != nil || parsedOrgID != orgID {
			continue
		}
		result = append(result, &entity.Membership{
			OrganizationID: parsedOrgID,
			UserID:         parsedUserID,
			Role:           role,
		})
	}
	return result, nil
}

func (m *removeMockMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	return nil
}

func (m *removeMockMembershipRepo) Remove(_ context.Context, orgID, userID uuid.UUID) error {
	m.removedOrg = orgID
	m.removedUser = userID
	return nil
}

type removeMockAuditLogRepo struct {
	recorded bool
}

func (m *removeMockAuditLogRepo) Record(context.Context, uuid.UUID, uuid.UUID, string, string, uuid.UUID, map[string]any) error {
	m.recorded = true
	return nil
}

func parseMembershipRoleKey(key string) (uuid.UUID, uuid.UUID, error) {
	parts := splitMembershipRoleKey(key)
	if len(parts) != 2 {
		return uuid.Nil, uuid.Nil, errors.New("invalid key")
	}
	orgID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	userID, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return orgID, userID, nil
}

func splitMembershipRoleKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == ':' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key}
}

func TestRemoveMemberLastOwner(t *testing.T) {
	memberships := &removeMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testOwnerID): entity.RoleOwner,
		},
	}
	auditLog := &removeMockAuditLogRepo{}
	uc := org.NewRemoveMemberUsecase(memberships, auditLog)

	err := uc.Execute(context.Background(), org.RemoveMemberInput{
		OrgID:        testOrgID,
		CallerID:     testOwnerID,
		TargetUserID: testOwnerID,
	})
	if !errors.Is(err, org.ErrLastOwner) {
		t.Fatalf("expected ErrLastOwner, got %v", err)
	}
	if memberships.removedUser != uuid.Nil {
		t.Fatal("expected member not to be removed")
	}
	if auditLog.recorded {
		t.Fatal("expected audit log not to be recorded")
	}
}

func TestRemoveMemberNonOwnerForbidden(t *testing.T) {
	memberships := &removeMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testMemberID): entity.RoleMember,
			membershipsRoleKey(testOrgID, testOwnerID):  entity.RoleOwner,
		},
	}
	auditLog := &removeMockAuditLogRepo{}
	uc := org.NewRemoveMemberUsecase(memberships, auditLog)

	err := uc.Execute(context.Background(), org.RemoveMemberInput{
		OrgID:        testOrgID,
		CallerID:     testMemberID,
		TargetUserID: testOwnerID,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if memberships.removedUser != uuid.Nil {
		t.Fatal("expected member not to be removed")
	}
}

func TestRemoveMemberHappyPath(t *testing.T) {
	memberships := &removeMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testOwnerID):  entity.RoleOwner,
			membershipsRoleKey(testOrgID, testMemberID): entity.RoleMember,
		},
	}
	auditLog := &removeMockAuditLogRepo{}
	uc := org.NewRemoveMemberUsecase(memberships, auditLog)

	err := uc.Execute(context.Background(), org.RemoveMemberInput{
		OrgID:        testOrgID,
		CallerID:     testOwnerID,
		TargetUserID: testMemberID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if memberships.removedOrg != testOrgID || memberships.removedUser != testMemberID {
		t.Fatalf("expected Remove(%s, %s), got (%s, %s)", testOrgID, testMemberID, memberships.removedOrg, memberships.removedUser)
	}
	if !auditLog.recorded {
		t.Fatal("expected audit log to be recorded")
	}
}
