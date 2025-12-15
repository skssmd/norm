package engine

import (
	"context"
	"errors"
	"path"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cacher defines the interface for caching strategies
type Cacher interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, pattern string) error // Delete by glob pattern (e.g., *user*)
}

// ============================================================
// Memory Cacher (In-Memory)
// ============================================================

type item struct {
	value     []byte
	expiresAt time.Time
}

// MemoryCacher implements Cacher using sync.Map
type MemoryCacher struct {
	items sync.Map
}

func NewMemoryCacher() *MemoryCacher {
	m := &MemoryCacher{}
	// Launch cleanup goroutine? 
	// For simplicity, we'll check expiry on Get for now, 
	// but a background cleanup is better for memory. 
	// Let's implement lazy expiry for now.
	return m
}

func (m *MemoryCacher) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.items.Load(key)
	if !ok {
		return nil, errors.New("key not found")
	}

	it, ok := val.(item)
	if !ok {
		return nil, errors.New("invalid cache item")
	}

	if time.Now().After(it.expiresAt) {
		m.items.Delete(key)
		return nil, errors.New("key expired")
	}

	return it.value, nil
}

func (m *MemoryCacher) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.items.Store(key, item{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	})
	return nil
}

func (m *MemoryCacher) Delete(ctx context.Context, pattern string) error {
	m.items.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			// Use path.Match for glob matching (*, ?)
			matched, err := path.Match(pattern, keyStr)
			if err == nil && matched {
				m.items.Delete(key)
			}
		}
		return true
	})
	return nil
}

// ============================================================
// Redis Cacher
// ============================================================

// RedisCacher implements Cacher using Redis
type RedisCacher struct {
	client *redis.Client
}

func NewRedisCacher(client *redis.Client) *RedisCacher {
	return &RedisCacher{client: client}
}

func (r *RedisCacher) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, errors.New("key not found")
	}
	return val, err
}

func (r *RedisCacher) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCacher) Delete(ctx context.Context, pattern string) error {
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		r.client.Del(ctx, iter.Val())
	}
	return iter.Err()
}
