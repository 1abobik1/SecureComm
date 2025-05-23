package handshake_service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/binary"
	"time"

	"github.com/sirupsen/logrus"
)

// parseSessionBlob парсит пакет вида: [timestamp(8 byte) || nonce(16 byte) || IV(16 byte) || ciphertext || tag(32 byte)]
// выдает чистые payload данные клиента
func (s *service) parseSessionBlob(ctx context.Context, clientID string, signature, blob []byte) ([]byte, error) {
	const op = "internal.service.parseSessionBlob"

	// парсим signature3 в r3, s3
	var sigDER der
	if _, err := asn1.Unmarshal(signature, &sigDER); err != nil {
		logrus.Errorf("%s: unmarshal sig3: %v", op, err)
		return nil, ErrInvalidPayload
	}

	clientECDSAPub, err := s.clientPubKeyStore.GetClientECDSAPub(ctx, clientID)
	if err != nil {
		logrus.Errorf("%s: fetch client pub: %v", op, err)
		return nil, err
	}

	// проверяем подпись ECDSA публичным ключем клиента: h3 = sha256(payload)
	h := sha256.Sum256(blob)
	if !ecdsa.Verify(clientECDSAPub, h[:], sigDER.R, sigDER.S) {
		return nil, ErrBadSignature
	}

	// проверяем длину всего blob = IV(16) + tag(32) + минимум 1 байт ciphertext
	if len(blob) < aes.BlockSize+sha256.Size+1 {
		logrus.Errorf("%s: invalid blob length %d", op, len(blob))
		return nil, ErrInvalidPayload
	}

	// раскладываем на IV, ciphertext, tag
	iv := blob[:aes.BlockSize]
	tagStart := len(blob) - sha256.Size
	ciphertext := blob[aes.BlockSize:tagStart]
	tag := blob[tagStart:]

	// получаем сессионный ключ
	kEnc, kMac, err := s.sessions.GetSessionKeys(ctx, clientID)
	if err != nil {
		logrus.Errorf("%s: invalid session for clientID %s", op, clientID)
		return nil, ErrInvalidSession
	}

	// проверяем HMAC(iv||ciphertext)
	mac := hmac.New(sha256.New, kMac)
	mac.Write(iv)
	mac.Write(ciphertext)
	if !hmac.Equal(mac.Sum(nil), tag) {
		logrus.Errorf("%s: bad MAC for clientID %s", op, clientID)
		return nil, ErrBadMAC
	}

	// AES-CBC расшифровка
	block, err := aes.NewCipher(kEnc)
	if err != nil {
		logrus.Errorf("%s: cipher init error: %v", op, err)
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	padded := make([]byte, len(ciphertext))
	mode.CryptBlocks(padded, ciphertext)

	// PKCS#7
	padLen := int(padded[len(padded)-1])
	if padLen <= 0 || padLen > aes.BlockSize {
		logrus.Errorf("%s: invalid padding length %d", op, padLen)
		return nil, ErrInvalidPayload
	}
	for _, b := range padded[len(padded)-padLen:] {
		if int(b) != padLen {
			logrus.Errorf("%s: invalid padding byte %d", op, b)
			return nil, ErrInvalidPayload
		}
	}
	plaintextBlob := padded[:len(padded)-padLen]

	// извлекаем метаданные из plaintextBlob
	if len(plaintextBlob) < 8+16 {
		logrus.Errorf("%s: metadata too short %d", op, len(plaintextBlob))
		return nil, ErrInvalidPayload
	}
	ts := int64(binary.BigEndian.Uint64(plaintextBlob[0:8]))
	nonce := plaintextBlob[8:24]
	userData := plaintextBlob[24:]

	now := time.Now().UnixMilli()

	// allowedWindow - задает время, насколько сообщение может быть запоздалым от клиента.
	// allowedWindow равен времени хранения nonce в redis
	allowedWindow := s.sesNonces.GetNonceTTL()

	diff := time.Duration(now-ts) * time.Millisecond
	if diff > allowedWindow || diff < -allowedWindow {
		logrus.Errorf("%s: stale timestamp (now=%d ts=%d diff=%v)", op, now, ts, diff)
		return nil, ErrStaleTimestamp
	}

	// проверяем и сохраняем nonce
	if s.sesNonces.Has(ctx, nonce) {
		logrus.Errorf("%s: replay detected for nonce %x", op, nonce)
		return nil, ErrReplayDetected
	}
	s.sesNonces.Add(ctx, nonce)

	// возвращает payload клиента
	return userData, nil
}
