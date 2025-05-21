package nonce_store

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// реализует NonceStore через Redis
type redisNonceStore struct {
	cli     *redis.Client
	ctx     context.Context
	ttl     time.Duration
	keyPref string
}

// addr — адрес redis, ttl — время хранения nonce
func NewRedisNonceStore(addr string, ttl time.Duration) *redisNonceStore {
	return &redisNonceStore{
		cli:     redis.NewClient(&redis.Options{Addr: addr}),
		ctx:     context.Background(),
		ttl:     ttl,
		keyPref: "nonce:",
	}
}

// проверяет, встречался ли такой nonce раньше
func (r *redisNonceStore) Has(ctx context.Context, nonce []byte) bool {
	const op = "location internal.repository.client_noncestore.Has"

	key := r.keyPref + hex.EncodeToString(nonce)
	exists, err := r.cli.Exists(r.ctx, key).Result()
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return true
	}
	return exists > 0
}

// сохраняет nonce с TTL, чтобы потом отбрасывать повторы
func (r *redisNonceStore) Add(ctx context.Context, nonce []byte) {
	key := r.keyPref + hex.EncodeToString(nonce)
	// SET NX с TTL
	r.cli.SetNX(r.ctx, key, "1", r.ttl)
}
