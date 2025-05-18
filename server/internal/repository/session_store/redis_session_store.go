package session_store

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type redisSessionStore struct {
	cli *redis.Client
	ctx context.Context
	ttl time.Duration
}

func NewRedisSessionStore(addr string, ttl time.Duration) *redisSessionStore {
	return &redisSessionStore{
		cli: redis.NewClient(&redis.Options{Addr: addr}),
		ctx: context.Background(),
		ttl: ttl,
	}
}

func (r *redisSessionStore) SaveSessionKeys(ctx context.Context, clientID string, kEnc, kMac []byte) error {
	// составим единый blob: kEnc||kMac
	blob := append(kEnc, kMac...)
	key := fmt.Sprintf("sess:%s", clientID)
	return r.cli.SetEX(ctx, key, blob, r.ttl).Err()
}

func (r *redisSessionStore) GetSessionKeys(ctx context.Context, clientID string) (kEnc []byte, kMac []byte, er error) {
	key := fmt.Sprintf("sess:%s", clientID)
	blob, err := r.cli.Get(ctx, key).Bytes()
	if err != nil {
		return nil, nil, err
	}
	if len(blob) != 64 {
		return nil, nil, fmt.Errorf("invalid session blob size")
	}
	return blob[:32], blob[32:], nil
}

func (r *redisSessionStore) DeleteSession(ctx context.Context, clientID string) error {
	return r.cli.Del(ctx, fmt.Sprintf("sess:%s", clientID)).Err()
}
