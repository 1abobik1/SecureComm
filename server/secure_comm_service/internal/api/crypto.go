package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

func encryptAESGCM(password string, data []byte) (ivB64, ctB64 string, err error) {
	key := sha256.Sum256([]byte(password))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return
	}
	iv := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, iv)
	ct := gcm.Seal(nil, iv, data, nil)
	ivB64 = base64.StdEncoding.EncodeToString(iv)
	ctB64 = base64.StdEncoding.EncodeToString(ct)
	return
}
