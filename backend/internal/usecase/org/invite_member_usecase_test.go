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

type inviteMockMembershipRepo struct {
	roles      map[string]string
	added      []*entity.Membership
	addErr     error
	updated    bool
}

func (m *inviteMockMembershipRepo) Add(_ context.Context, membership *entity.Membership) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.added = append(m.added, membership)
	return nil
}

func (m *inviteMockMembershipRepo) GetRole(_ context.Context, orgID, userID uuid.UUID) (string, error) {
	if m.roles == nil {
		return "", domain.ErrNotFound
	}
	role, ok := m.roles[membershipsRoleKey(orgID, userID)]
	if !ok {
		return "", domain.ErrNotFound
	}
	return role, nil
}

func (m *inviteMockMembershipRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Membership, error) {
	return nil, nil
}

func (m *inviteMockMembershipRepo) UpdateRole(context.Context, uuid.UUID, uuid.UUID, string) error {
	m.updated = true
	return nil
}

func (m *inviteMockMembershipRepo) Remove(context.Context, uuid.UUID, uuid.UUID) error { return nil }

func TestInviteMemberNonOwnerForbidden(t *testing.T) {
	targetID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	memberships := &inviteMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testMemberID): entity.RoleMember,
		},
	}
	uc := org.NewInviteMemberUsecase(memberships)

	err := uc.Execute(context.Background(), org.InviteMemberInput{
		OrgID:        testOrgID,
		CallerID:     testMemberID,
		TargetUserID: targetID,
		Role:         entity.RoleMember,
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
	if len(memberships.added) != 0 {
		t.Fatal("expected no membership to be added")
	}
}

func TestInviteMemberValidMemberRole(t *testing.T) {
	targetID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	memberships := &inviteMockMembershipRepo{
		roles: map[string]string{
			membershipsRoleKey(testOrgID, testOwnerID): entity.RoleOwner,
		},
	}
	uc := org.NewInviteMemberUsecase(memberships)

	err := uc.Execute(context.Background(), org.InviteMemberInput{
		OrgID:        testOrgID,
		CallerID:     testOwnerID,
		TargetUserID: targetID,
		Role:         entity.RoleMember,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(memberships.added) != 1 {
		t.Fatal("expected membership to be added")
	}
	membership := memberships.added[0]
	if membership.OrganizationID != testOrgID {
		t.Fatalf("expected org %s, got %s", testOrgID, membership.OrganizationID)
	}
	if membership.UserID != targetID {
		t.Fatalf("expected user %s, got %s", targetID, membership.UserID)
	}
	if membership.Role != entity.RoleMember {
		t.Fatalf("expected member role, got %s", membership.Role)
	}
}
