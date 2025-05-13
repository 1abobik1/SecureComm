package client_keystore

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// хранит публичные ключи клиентов в Redis по clientID
type redisClientPubKeyStore struct {
	redis *redis.Client
	ctx   context.Context
}

func NewRedisClientPubKeyStore(addr string) *redisClientPubKeyStore {
	cli := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &redisClientPubKeyStore{
		redis: cli,
		ctx:   context.Background(),
	}
}

// сохраняет в Redis пару ключей клиента под префиксом client:{clientID}
func (r *redisClientPubKeyStore) SaveClientKeys(clientID string, rsaPubDER, ecdsaPubDER []byte) error {
	
	keyRSA := fmt.Sprintf("client:%s:rsa_pub", clientID)
	keyECDSA := fmt.Sprintf("client:%s:ecdsa_pub", clientID)

	// для хранения кодируем DER→Base64, чтобы не зависеть от бинарных особенностей Redis
	if err := r.redis.Set(r.ctx, keyRSA,
		base64.StdEncoding.EncodeToString(rsaPubDER), 0).Err(); err != nil {
		return fmt.Errorf("redis save rsa_pub: %w", err)
	}
	if err := r.redis.Set(r.ctx, keyECDSA,
		base64.StdEncoding.EncodeToString(ecdsaPubDER), 0).Err(); err != nil {
		return fmt.Errorf("redis save ecdsa_pub: %w", err)
	}
	return nil
}

// возвращает raw DER-байты RSA-публичного ключа клиента
func (r *redisClientPubKeyStore) GetClientRSAPub(clientID string) ([]byte, error) {
	key := fmt.Sprintf("client:%s:rsa_pub", clientID)
	b64, err := r.redis.Get(r.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis get rsa_pub: %w", err)
	}
	return base64.StdEncoding.DecodeString(b64)
}

// возвращает объект *ecdsa.PublicKey клиента
func (r *redisClientPubKeyStore) GetClientECDSAPub(clientID string) (*ecdsa.PublicKey, error) {
	key := fmt.Sprintf("client:%s:ecdsa_pub", clientID)
	b64, err := r.redis.Get(r.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis get ecdsa_pub: %w", err)
	}

	der, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode ecdsa_pub: %w", err)
	}

	// Парсим DER-PKIX PublicKey → ecdsa.PublicKey
	pubIfc, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse ecdsa pub DER: %w", err)
	}
	pubECDSA, ok := pubIfc.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("unexpected public key type, want ECDSA")
	}
	return pubECDSA, nil
}
