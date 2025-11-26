package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"movieexample.com/metadata/pkg/model"
)

type fallbackRepository interface {
	Get(ctx context.Context, id string) (*model.Metadata, error)
	Put(ctx context.Context, id string, metadata *model.Metadata) error
}

// Repository defines a Redis-based movie metadata cache repository.
type Repository struct {
	client   *redis.Client
	fallback fallbackRepository
	ttl      time.Duration
}

// New creates a new Redis-based cache repository.
func New(fallback fallbackRepository) *Repository {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	return &Repository{client: rdb, fallback: fallback, ttl: 4 * time.Hour}
}

// Get retrieves movie metadata for by movie id.
func (r *Repository) Get(ctx context.Context, id string) (*model.Metadata, error) {
	key := fmt.Sprintf("metadata:%s", id)
	lockKey := fmt.Sprintf("lock:%s", key)
	lockTTL := 5 * time.Second

	val, err := r.client.Get(ctx, key).Result()
	if err == nil {
		var res model.Metadata
		json.Unmarshal([]byte(val), &res)
		return &res, nil
	}

	acquired, err := r.client.SetArgs(ctx, lockKey, 1, redis.SetArgs{Mode: "NX", TTL: lockTTL}).Result()
	if err == nil && acquired == "OK" {
		data, err := r.fallback.Get(ctx, id)
		if err != nil {
			r.client.Del(ctx, lockKey)
			return nil, err
		}

		marshaled, err := json.Marshal(data)
		if err != nil {
			r.client.Del(ctx, lockKey)
			return nil, err
		}
		r.client.Set(ctx, key, marshaled, r.ttl)

		r.client.Del(ctx, lockKey)
		return data, nil
	}

	time.Sleep(100 * time.Millisecond)

	return r.Get(ctx, id)
}

// Put adds movie metadata for a given movie id.
func (r *Repository) Put(ctx context.Context, id string, metadata *model.Metadata) error {
	if err := r.fallback.Put(ctx, id, metadata); err != nil {
		return err
	}

	key := fmt.Sprintf("metadata:%s", id)
	vb, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, vb, r.ttl).Err()
}
