package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
)

// DecryptWithSession расшифровывает пакет [IV||ciphertext||tag] с использованием сессионных ключей.
// Возвращает plaintext или одну из ErrInvalidSession, ErrInvalidPayload, ErrBadMAC.
func (s *service) DecryptWithSession(ctx context.Context, clientID string, data []byte) ([]byte, error) {
	// получаем ключи из хранилища
	kEnc, kMac, err := s.sessions.GetSessionKeys(ctx, clientID)
	if err != nil {
		return nil, ErrInvalidSession
	}

	// проверяем длину: минимум IV (16) + HMAC (32)
	if len(data) < aes.BlockSize+sha256.Size {
		return nil, ErrInvalidPayload
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize : len(data)-sha256.Size]
	tag := data[len(data)-sha256.Size:]

	// проверяем HMAC-SHA256(iv||ciphertext)
	mac := hmac.New(sha256.New, kMac)
	mac.Write(iv)
	mac.Write(ciphertext)
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, tag) {
		return nil, ErrBadMAC
	}

	// расшифровка AES-256-CBC
	block, err := aes.NewCipher(kEnc)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	padded := make([]byte, len(ciphertext))
	mode.CryptBlocks(padded, ciphertext)

	// убираем PKCS#7 паддинг
	padLen := int(padded[len(padded)-1])
	if padLen <= 0 || padLen > aes.BlockSize {
		return nil, ErrInvalidPayload
	}
	for _, v := range padded[len(padded)-padLen:] {
		if int(v) != padLen {
			return nil, ErrInvalidPayload
		}
	}
	plaintext := padded[:len(padded)-padLen]
	return plaintext, nil
}
