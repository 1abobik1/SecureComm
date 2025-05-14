package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"errors"

	"github.com/sirupsen/logrus"
)

func (s *service) Init(ctx context.Context, clientID string, clientRSAPubDER, clientECDSAPubDER, nonce1, sig1 []byte) (serverRSA, serverECDSA, nonce2, signature2 []byte, err error) {
	const op = "location internal.service.handshake_init.Init"

	// replay-защита
	if s.nonces.Has(ctx, nonce1) {
		return nil, nil, nil, nil, ErrReplayDetected
	}
	s.nonces.Add(ctx, nonce1)

	// получаем серверные ключи
	_, rsaPubS, ecdsaPrivS, ecdsaPubS := s.servKeysStore.GetServerKeys()

	// импортируем публичный ECDSA-ключ клиента из DER
	pubIfc, err := x509.ParsePKIXPublicKey(clientECDSAPubDER)
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("invalid ECDSA public key format")
	}
	pubClientECDSA, ok := pubIfc.(*ecdsa.PublicKey)
	if !ok {
		return nil, nil, nil, nil, errors.New("not an ECDSA public key")
	}

	// проверка подписи клиента
	// h1 = SHA256(clientRSAPubDER ∥ clientECDSAPubDER ∥ nonce1)
	totalLenH1 := len(clientRSAPubDER) + len(clientECDSAPubDER) + len(nonce1)
	dataH1 := make([]byte, 0, totalLenH1)

	dataH1 = append(dataH1, clientRSAPubDER...)
	dataH1 = append(dataH1, clientECDSAPubDER...)
	dataH1 = append(dataH1, nonce1...)

	h1 := sha256.Sum256(dataH1)

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

	// сохраняем публичные ключи клиента
	if err := s.clientPubKeyStore.SaveClientKeys(ctx, clientID, clientRSAPubDER, clientECDSAPubDER); err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("error to save client keys in redis")
	}

	// генерируем nonce2
	nonce2 = make([]byte, 8)
	if _, err = rand.Read(nonce2); err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("cannot generate nonce2")
	}

	// подписываем ответ: dataH2 = rsaPubS ∥ ecdsaPubS ∥ nonce2 ∥ nonce1 ∥ clientID
	totalLenH2 := len(rsaPubS) + len(ecdsaPubS) + len(nonce2) + len(nonce1) + len(clientID)
	dataH2 := make([]byte, 0, totalLenH2)

	dataH2 = append(dataH2, rsaPubS...)
	dataH2 = append(dataH2, ecdsaPubS...)
	dataH2 = append(dataH2, nonce2...)
	dataH2 = append(dataH2, nonce1...)
	dataH2 = append(dataH2, clientID...)

	h2 := sha256.Sum256(dataH2)

	// процесс продписи приватным ключем сервера
	r2, s2, err := ecdsa.Sign(rand.Reader, ecdsaPrivS, h2[:])
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("failed to sign response")
	}
	// кодируем в der байты
	signature2, err = asn1.Marshal(der{R: r2, S: s2})
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, nil, nil, nil, errors.New("failed to marshal signature")
	}

	return rsaPubS, ecdsaPubS, nonce2, signature2, nil
}

func (s *service) ComputeFingerprint(ctx context.Context, rsaPub, ecdsaPub []byte) string {
	h := sha256.Sum256(append(rsaPub, ecdsaPub...))
	return hex.EncodeToString(h[:]) // 64-символьный hex
}

// Finalize расшифровывает и проверяет подписанное RSA-OAEP сообщение,
// извлекает key_session и nonce3, проверяет ECDSA-подпись, и возвращает nil в случае успеха.
func (s *service) Finalize(ctx context.Context, clientID string, encrypted []byte) (signature4 []byte, err error) {
	const op = "internal.service.handshake.Finalize"

	rsaPrivS, _, ecdsaPrivS, _ := s.servKeysStore.GetServerKeys()

	// RSA-OAEP расшифровка
	// toEncrypt = payload ∥ signature3(DER)
	toEncrypt, err := rsaPrivS.Decrypt(
		nil,
		encrypted,
		&rsa.OAEPOptions{Hash: crypto.SHA256},
	)
	if err != nil {
		logrus.Errorf("%s: decrypt error: %v", op, err)
		return nil, ErrInvalidPayload
	}

	// разбираем payload и sig3
	// payloadLen = 32 + 8 + 8 = 48 байт
	if len(toEncrypt) < 48 {
		return nil, ErrInvalidPayload
	}
	payload := toEncrypt[:48]
	sig3DER := toEncrypt[48:]

	// парсим signature3 в r3, s3
	var sig3 der
	if _, err := asn1.Unmarshal(sig3DER, &sig3); err != nil {
		logrus.Errorf("%s: unmarshal sig3: %v", op, err)
		return nil, ErrInvalidPayload
	}

	clientECDSAPub, err := s.clientPubKeyStore.GetClientECDSAPub(ctx, clientID)
	if err != nil {
		logrus.Errorf("%s: fetch client pub: %v", op, err)
		return nil, err
	}

	// проверяем подпись ECDSA публичным ключем клиента: h3 = sha256(payload)
	h3 := sha256.Sum256(payload)
	if !ecdsa.Verify(clientECDSAPub, h3[:], sig3.R, sig3.S) {
		return nil, ErrBadSignature
	}

	// разбираем payload на компоненты
	ks := payload[:32]
	nonce3 := payload[32:40]
	nonce2 := payload[40:48]

	// replay–защита nonce3
	if s.nonces.Has(ctx, nonce3) {
		return nil, ErrReplayDetected
	}
	s.nonces.Add(ctx, nonce3)

	kEnc := hkdfSha256(ks, []byte("enc"))
	kMac := hkdfSha256(ks, []byte("mac"))

	err = s.sessions.SaveSessionKeys(ctx, clientID, kEnc, kMac)
	if err != nil {
		return nil, err
	}

	// подписываем те же данные от клиента, но уже приватным ключем сервера
	// подписываем ответ: data = ks ∥ nonce3 ∥ nonce2
	totalLen := len(ks) + len(nonce3) + len(nonce2)
	buf := make([]byte, 0, totalLen)

	buf = append(buf, ks...)
	buf = append(buf, nonce3...)
	buf = append(buf, nonce2...)

	h4 := sha256.Sum256(buf)

	// процесс продписи приватным ключем сервера
	r4, s4, err := ecdsa.Sign(rand.Reader, ecdsaPrivS, h4[:])
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, errors.New("failed to sign response")
	}
	// кодируем в der байты
	signature4, err = asn1.Marshal(der{R: r4, S: s4})
	if err != nil {
		logrus.Errorf("%s: %v", op, err)
		return nil, errors.New("failed to marshal signature")
	}

	// ответ клиенту. хеш h4 = SHA256(Ks || nonce3 || nonce2) подписанный приватным ключем сервера
	return signature4, nil
}
