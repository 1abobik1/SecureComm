package client

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
)

// ------------------------------
// Тип Session в client/handshake.go
// ------------------------------
type Session struct {
	ClientID    string
	ECDSAPriv   *ecdsa.PrivateKey
	Ks          []byte
	KEnc        []byte
	KMac        []byte
	TestURL     string
	AccessToken string
}

// NewSession
func NewSession(clientID string, ecdsaPriv *ecdsa.PrivateKey, ks []byte, testURL, accessToken string) *Session {
	mac := hmac.New(sha256.New, ks)
	mac.Write([]byte("mac"))
	kmac := mac.Sum(nil)
	enc := hmac.New(sha256.New, ks)
	enc.Write([]byte("enc"))
	kenc := enc.Sum(nil)

	return &Session{
		ClientID:    clientID,
		ECDSAPriv:   ecdsaPriv,
		Ks:          ks,
		KEnc:        kenc,
		KMac:        kmac,
		TestURL:     testURL,
		AccessToken: accessToken,
	}
}
