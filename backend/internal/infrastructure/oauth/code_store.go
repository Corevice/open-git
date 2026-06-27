package oauth

import (
	"context"
	"errors"
	"sync"
	"time"

	authUC "github.com/open-git/backend/internal/usecase/auth"
	"github.com/redis/go-redis/v9"
)

type oauthCodeEntry struct {
	value   string
	expires time.Time
}

type MemoryCodeStore struct {
	mu   sync.Mutex
	data map[string]oauthCodeEntry
}

func NewMemoryCodeStore() *MemoryCodeStore {
	return &MemoryCodeStore{data: make(map[string]oauthCodeEntry)}
}

func (s *MemoryCodeStore) Set(_ context.Context, key, value string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked()
	s.data[key] = oauthCodeEntry{
		value:   value,
		expires: time.Now().Add(ttl),
	}
	return nil
}

func (s *MemoryCodeStore) GetDel(_ context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.data[key]
	if !ok {
		return "", errors.New("oauth code not found")
	}
	if time.Now().After(entry.expires) {
		delete(s.data, key)
		return "", errors.New("oauth code expired")
	}
	delete(s.data, key)
	return entry.value, nil
}

func (s *MemoryCodeStore) purgeExpiredLocked() {
	now := time.Now()
	for key, entry := range s.data {
		if now.After(entry.expires) {
			delete(s.data, key)
		}
	}
}

type RedisCodeStore struct {
	Client *redis.Client
}

func NewRedisCodeStore(addr string) *RedisCodeStore {
	return &RedisCodeStore{Client: redis.NewClient(&redis.Options{Addr: addr})}
}

func (s *RedisCodeStore) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.Client.Set(ctx, key, value, ttl).Err()
}

func (s *RedisCodeStore) GetDel(ctx context.Context, key string) (string, error) {
	value, err := s.Client.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", errors.New("oauth code not found")
	}
	return value, err
}

func (s *RedisCodeStore) Close() error {
	return s.Client.Close()
}

func NewCodeStore(redisAddr string) (authUC.OAuthCodeStore, func() error) {
	if redisAddr != "" {
		store := NewRedisCodeStore(redisAddr)
		return store, store.Close
	}
	return NewMemoryCodeStore(), func() error { return nil }
}
