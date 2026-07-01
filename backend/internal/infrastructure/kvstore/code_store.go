// Package kvstore provides small key-value stores with TTL semantics used for
// short-lived server-side state such as OAuth authorization codes.
package kvstore

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// InMemoryTTLStore is a process-local TTL key-value store. It backs the OAuth
// authorization-code flow on single-process deployments (e.g. the SQLite
// setup) where no Redis is configured. Codes are short-lived (minutes) and
// consumed once, so the memory footprint stays negligible; expired entries are
// also swept lazily on writes.
type InMemoryTTLStore struct {
	mu    sync.Mutex
	items map[string]inMemoryItem
}

type inMemoryItem struct {
	value     string
	expiresAt time.Time
}

func NewInMemoryTTLStore() *InMemoryTTLStore {
	return &InMemoryTTLStore{items: make(map[string]inMemoryItem)}
}

func (s *InMemoryTTLStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, item := range s.items {
		if now.After(item.expiresAt) {
			delete(s.items, k)
		}
	}
	s.items[key] = inMemoryItem{value: value, expiresAt: now.Add(ttl)}
	return nil
}

// GetDel returns the value for key and removes it atomically. A missing or
// expired key yields ("", nil), matching the redis GetDel-with-redis.Nil
// contract the callers rely on.
func (s *InMemoryTTLStore) GetDel(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[key]
	if !ok {
		return "", nil
	}
	delete(s.items, key)
	if time.Now().After(item.expiresAt) {
		return "", nil
	}
	return item.value, nil
}

// RedisTTLStore backs the same contract with Redis, for multi-replica
// deployments.
type RedisTTLStore struct {
	client *redis.Client
}

func NewRedisTTLStore(client *redis.Client) *RedisTTLStore {
	return &RedisTTLStore{client: client}
}

func (s *RedisTTLStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *RedisTTLStore) GetDel(ctx context.Context, key string) (string, error) {
	value, err := s.client.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}
