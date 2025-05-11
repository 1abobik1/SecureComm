package service

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/sirupsen/logrus"
)

var (
	ErrReplayDetected = errors.New("replay detected")
)

// NonceStore — хранит использованные nonces, чтобы отсеять replay
type NonceStore interface {
	Has(nonce []byte) bool
	Add(nonce []byte)
}

// KeyStore — даёт доступ к парам ключей сервера
type ServerKeyStore interface {
	GetServerKeys() (*rsa.PrivateKey, []byte, *ecdsa.PrivateKey, []byte)
}

// ClientPubKeyStore - хранит и отдает публичные ключи пользователей в REDIS
type ClientPubKeyStore interface {
	// SaveClientKeys сохраняет публичные ключи клиента по clientID
	SaveClientKeys(clientID string, rsaPubDER, ecdsaPubDER []byte) error
	
	GetClientRSAPub(clientID string) ([]byte, error)
	GetClientECDSAPub(clientID string) (*ecdsa.PublicKey, error)
}

type service struct {
	nonces            NonceStore
	servKeysStore     ServerKeyStore
	clientPubKeyStore ClientPubKeyStore
}

func NewService(nonces NonceStore, servKeysStore ServerKeyStore, clientPubKeyStore ClientPubKeyStore) *service {
	return &service{
		nonces:            nonces,
		servKeysStore:     servKeysStore,
		clientPubKeyStore: clientPubKeyStore,
	}
}

func (s *service) Init(clientID string, clientRSAPubDER, clientECDSAPubDER, nonce1, sig1 []byte) (serverRSA, serverECDSA, nonce2, signature2 []byte, err error) {
	const op = "location internal.service.handshake_init.Init"

	// 1) Replay-защита
	if s.nonces.Has(nonce1) {
		return nil, nil, nil, nil, ErrReplayDetected
	}
	s.nonces.Add(nonce1)

	// 2) Получаем серверные ключи
	_, rsaPubS, ecdsaPrivS, ecdsaPubS := s.servKeysStore.GetServerKeys()

	// 3) Импортируем публичный ECDSA-ключ клиента из DER
	pubIfc, err := x509.ParsePKIXPublicKey(clientECDSAPubDER)
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("invalid ECDSA public key format")
	}
	pubClientECDSA, ok := pubIfc.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, nil, nil, errors.New("not an ECDSA public key")
	}

	// 4) Проверяем подпись клиента
	//    h1 = SHA256(clientRSAPubDER ∥ clientECDSAPubDER ∥ nonce1)
	h1 := sha256.Sum256(append(append(clientRSAPubDER, clientECDSAPubDER...), nonce1...))

	// разбираем sig1 DER → {R,S}
	var clientSig der
	if _, err := asn1.Unmarshal(sig1, &clientSig); err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("invalid signature format")
	}

	// верификация
	if !ecdsa.Verify(pubClientECDSA, h1[:], clientSig.R, clientSig.S) {
		return nil, nil, nil, nil, errors.New("signature verification failed")
	}

	// 5) Сохраняем публичные ключи клиента
	if err := s.clientPubKeyStore.SaveClientKeys(clientID, clientRSAPubDER, clientECDSAPubDER); err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("error to save client keys in redis")
	}

	// 6) Генерируем nonce2
	nonce2 = make([]byte, 8)
	if _, err = rand.Read(nonce2); err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("cannot generate nonce2")
	}

	// 7) Подписываем ответ: data2 = rsaPubS ∥ ecdsaPubS ∥ nonce2 ∥ nonce1
	data2 := append(append(append(rsaPubS, ecdsaPubS...), nonce2...), nonce1...)
	h2 := sha256.Sum256(data2)
	r2, s2, err := ecdsa.Sign(rand.Reader, ecdsaPrivS, h2[:])
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("failed to sign response")
	}
	signature2, err = asn1.Marshal(der{R: r2, S: s2})
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("failed to marshal signature")
	}

	// 8) Возвращаем ответные данные
	return rsaPubS, ecdsaPubS, nonce2, signature2, nil
}

func (s *service) ComputeFingerprint(rsaPub, ecdsaPub []byte) string {
	h := sha256.Sum256(append(rsaPub, ecdsaPub...))
	return hex.EncodeToString(h[:]) // 64-символьный hex
}

func (s *service) Finalize(encrypted []byte) (signature4 []byte, err error) {
	panic("some err")
}

type der struct {
	R *big.Int
	S *big.Int
}
