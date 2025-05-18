package nonce_store

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// RedisSessionNonceStore хранит session‑nonce для replay‑защиты сообщений.
type redisSessionNonceStore struct {
	cli     *redis.Client
	ctx     context.Context
	ttl     time.Duration
	keyPref string
}

// NewRedisSessionNonceStore создаёт стор с префиксом и TTL.
func NewRedisSessionNonceStore(addr string, ttl time.Duration) *redisSessionNonceStore {
	return &redisSessionNonceStore{
		cli:     redis.NewClient(&redis.Options{Addr: addr}),
		ctx:     context.Background(),
		ttl:     ttl,
		keyPref: "sessnonce:", // префикс
	}
}

// возвращает время в int
func (r *redisSessionNonceStore) GetNonceTTL() time.Duration {
	return r.ttl
}

// Has проверяет, встречался ли уже nonce в этом окне.
func (r *redisSessionNonceStore) Has(ctx context.Context, nonce []byte) bool {
	key := r.keyPref + hex.EncodeToString(nonce)
	exists, err := r.cli.Exists(r.ctx, key).Result()
	if err != nil {
		logrus.Errorf("sessionNonceStore.Has: %v", err)
		return true
	}
	return exists > 0
}

// Add сохраняет nonce с NX+TTL.
func (r *redisSessionNonceStore) Add(ctx context.Context, nonce []byte) {
	key := r.keyPref + hex.EncodeToString(nonce)
	r.cli.SetNX(r.ctx, key, "1", r.ttl)
}
