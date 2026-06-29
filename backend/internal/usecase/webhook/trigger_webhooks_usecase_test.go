package webhook_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
	"github.com/open-git/backend/internal/infrastructure/queue"
	webhookusecase "github.com/open-git/backend/internal/usecase/webhook"
)

type mockTriggerWebhookRepo struct {
	hooks []*entity.Webhook
	err   error
}

func (m *mockTriggerWebhookRepo) Create(context.Context, *entity.Webhook) error { return nil }
func (m *mockTriggerWebhookRepo) GetByID(context.Context, uuid.UUID, uuid.UUID) (*entity.Webhook, error) {
	return nil, nil
}
func (m *mockTriggerWebhookRepo) ListByRepo(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}
func (m *mockTriggerWebhookRepo) ListByOrg(context.Context, uuid.UUID, int, int) ([]*entity.Webhook, int64, error) {
	return nil, 0, nil
}
func (m *mockTriggerWebhookRepo) Update(context.Context, *entity.Webhook) error { return nil }
func (m *mockTriggerWebhookRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (m *mockTriggerWebhookRepo) ListActiveByRepoAndEvent(_ context.Context, _, _ uuid.UUID, _ string) ([]*entity.Webhook, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hooks, nil
}

var _ domainrepo.IWebhookRepository = (*mockTriggerWebhookRepo)(nil)

type mockTriggerDeliveryRepo struct {
	created []*entity.WebhookDelivery
}

func (m *mockTriggerDeliveryRepo) Create(_ context.Context, delivery *entity.WebhookDelivery) error {
	copyDelivery := *delivery
	m.created = append(m.created, &copyDelivery)
	return nil
}

func (m *mockTriggerDeliveryRepo) GetByID(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*entity.WebhookDelivery, error) {
	return nil, nil
}
func (m *mockTriggerDeliveryRepo) ListByWebhook(context.Context, uuid.UUID, uuid.UUID, int, int) ([]*entity.WebhookDelivery, int64, error) {
	return nil, 0, nil
}
func (m *mockTriggerDeliveryRepo) UpdateStatus(context.Context, uuid.UUID, string, *int, map[string][]string, *string, *int, time.Time) error {
	return nil
}

var _ domainrepo.IWebhookDeliveryRepository = (*mockTriggerDeliveryRepo)(nil)

type mockTaskEnqueuer struct {
	tasks []*asynq.Task
}

func (m *mockTaskEnqueuer) EnqueueContext(_ context.Context, task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	m.tasks = append(m.tasks, task)
	return &asynq.TaskInfo{ID: uuid.New().String()}, nil
}

func TestTriggerWebhooksUsecaseZeroMatchingWebhooks(t *testing.T) {
	webhookRepo := &mockTriggerWebhookRepo{hooks: nil}
	deliveryRepo := &mockTriggerDeliveryRepo{}
	enqueuer := &mockTaskEnqueuer{}

	uc := webhookusecase.NewTriggerWebhooksUsecase(webhookRepo, deliveryRepo, enqueuer)
	err := uc.Execute(context.Background(), webhookusecase.TriggerWebhooksInput{
		OrgID:   uuid.New(),
		RepoID:  uuid.New(),
		Event:   "push",
		Payload: []byte(`{"ref":"refs/heads/main"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(deliveryRepo.created) != 0 {
		t.Fatalf("deliveryRepo.Create calls = %d, want 0", len(deliveryRepo.created))
	}
	if len(enqueuer.tasks) != 0 {
		t.Fatalf("enqueue calls = %d, want 0", len(enqueuer.tasks))
	}
}

func TestTriggerWebhooksUsecaseTwoMatchingWebhooks(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	hookOneID := uuid.New()
	hookTwoID := uuid.New()

	webhookRepo := &mockTriggerWebhookRepo{
		hooks: []*entity.Webhook{
			{
				ID:             hookOneID,
				OrganizationID: orgID,
				ContentType:    entity.ContentTypeJSON,
				Active:         true,
			},
			{
				ID:             hookTwoID,
				OrganizationID: orgID,
				ContentType:    entity.ContentTypeForm,
				Active:         true,
			},
		},
	}
	deliveryRepo := &mockTriggerDeliveryRepo{}
	enqueuer := &mockTaskEnqueuer{}
	payload := []byte(`{"ref":"refs/heads/main"}`)

	uc := webhookusecase.NewTriggerWebhooksUsecase(webhookRepo, deliveryRepo, enqueuer)
	err := uc.Execute(context.Background(), webhookusecase.TriggerWebhooksInput{
		OrgID:   orgID,
		RepoID:  repoID,
		Event:   "push",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(deliveryRepo.created) != 2 {
		t.Fatalf("deliveryRepo.Create calls = %d, want 2", len(deliveryRepo.created))
	}
	if len(enqueuer.tasks) != 2 {
		t.Fatalf("enqueue calls = %d, want 2", len(enqueuer.tasks))
	}

	deliveryIDs := make(map[string]struct{})
	for i, delivery := range deliveryRepo.created {
		if delivery.Status != entity.StatusPending {
			t.Fatalf("delivery[%d].Status = %q, want pending", i, delivery.Status)
		}
		if delivery.OrganizationID != orgID {
			t.Fatalf("delivery[%d].OrganizationID = %v, want %v", i, delivery.OrganizationID, orgID)
		}
		if delivery.Event != "push" {
			t.Fatalf("delivery[%d].Event = %q, want push", i, delivery.Event)
		}
		if delivery.RequestBody != string(payload) {
			t.Fatalf("delivery[%d].RequestBody = %q, want %q", i, delivery.RequestBody, string(payload))
		}
		deliveryIDs[delivery.ID.String()] = struct{}{}
	}
	if len(deliveryIDs) != 2 {
		t.Fatal("expected distinct delivery IDs")
	}

	taskDeliveryIDs := make(map[string]struct{})
	for i, task := range enqueuer.tasks {
		if task.Type() != queue.TypeWebhookDeliver {
			t.Fatalf("task[%d].Type = %q, want %q", i, task.Type(), queue.TypeWebhookDeliver)
		}
		var taskPayload queue.WebhookDeliveryPayload
		if err := json.Unmarshal(task.Payload(), &taskPayload); err != nil {
			t.Fatalf("unmarshal task[%d] payload: %v", i, err)
		}
		if taskPayload.OrganizationID != orgID.String() {
			t.Fatalf("task[%d].OrganizationID = %q, want %q", i, taskPayload.OrganizationID, orgID.String())
		}
		if taskPayload.Event != "push" {
			t.Fatalf("task[%d].Event = %q, want push", i, taskPayload.Event)
		}
		if taskPayload.ContentType == "" {
			t.Fatalf("task[%d].ContentType is empty", i)
		}
		taskDeliveryIDs[taskPayload.DeliveryID] = struct{}{}
	}
	if len(taskDeliveryIDs) != 2 {
		t.Fatal("expected distinct delivery IDs in enqueued tasks")
	}
}
