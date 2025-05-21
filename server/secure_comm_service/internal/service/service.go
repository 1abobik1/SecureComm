package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"time"
)

var (
	ErrReplayDetected = errors.New("replay detected")
	ErrInvalidPayload = errors.New("invalid encrypted payload")
	ErrBadSignature   = errors.New("ECDSA signature verification failed")
	ErrBadMAC         = errors.New("message authentication failed")
	ErrInvalidSession = errors.New("invalid session or keys")
	ErrStaleTimestamp = errors.New("stale timestamp")
)

// хранит использованные nonces, чтобы отсеять replay
type HandshakeNoncesStore interface {
	Has(ctx context.Context, nonce []byte) bool
	Add(ctx context.Context, nonce []byte)
}

type SessioneNoncesStore interface {
	Has(ctx context.Context, nonce []byte) bool
	Add(ctx context.Context, nonce []byte)
	GetNonceTTL() time.Duration
}

// даёт доступ к парам ключей сервера
type ServerKeyStore interface {
	GetServerKeys() (*rsa.PrivateKey, []byte, *ecdsa.PrivateKey, []byte)
}

// хранит и отдает публичные ключи пользователей в REDIS
type ClientPubKeyStore interface {
	// сохраняет публичные ключи клиента по clientID
	SaveClientKeys(ctx context.Context, clientID string, rsaPubDER, ecdsaPubDER []byte) error
	GetClientRSAPub(ctx context.Context, clientID string) ([]byte, error)
	GetClientECDSAPub(ctx context.Context, clientID string) (*ecdsa.PublicKey, error)
}

type SessionStore interface {
	SaveSessionKeys(ctx context.Context, clientID string, kEnc, kMac []byte) error
	GetSessionKeys(ctx context.Context, clientID string) (kEnc, kMac []byte, err error)
	DeleteSession(ctx context.Context, clientID string) error
}

type service struct {
	hsNonces          HandshakeNoncesStore
	sesNonces         SessioneNoncesStore
	servKeysStore     ServerKeyStore
	clientPubKeyStore ClientPubKeyStore
	sessions          SessionStore
}

func NewService(hsNonces HandshakeNoncesStore, sesNonces SessioneNoncesStore, servKeysStore ServerKeyStore, clientPubKeyStore ClientPubKeyStore, sessionStore SessionStore) *service {
	return &service{
		hsNonces:          hsNonces,
		sesNonces:         sesNonces,
		servKeysStore:     servKeysStore,
		clientPubKeyStore: clientPubKeyStore,
		sessions:          sessionStore,
	}
}
