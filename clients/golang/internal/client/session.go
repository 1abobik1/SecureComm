package client

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"example_client/internal/crypto_utils"
	"fmt"
	"net/http"
	"time"
)

type Session struct {
	ClientID    string
	ECDSAPriv   *ecdsa.PrivateKey
	Ks          []byte
	KEnc        []byte
	KMac        []byte
	TestURL     string
	AccessToken string
}

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

// Session Test (с Authorization)
func (s *Session) DoSessionTest(plaintext string) error {
	// timestamp (8 байт)
	ts := time.Now().UnixMilli()
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(ts))

	// nonce (16 байт)
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	// собираем blob = timestamp||nonce||plaintext
	blob := append(append(timestamp, nonce...), []byte(plaintext)...)

	// iv (16 байт)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}

	block, _ := aes.NewCipher(s.KEnc)
	padded := crypto_utils.Pkcs7Pad(blob, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	mac := hmac.New(sha256.New, s.KMac)
	mac.Write(iv)
	mac.Write(ciphertext)
	tag := mac.Sum(nil)

	pkg := append(append(iv, ciphertext...), tag...)
	b64 := base64.StdEncoding.EncodeToString(pkg)

	reqBody := map[string]string{
		"encrypted_message": b64,
		"client_signature":  crypto_utils.MustSignPayloadECDSA(s.ECDSAPriv, pkg),
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.TestURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.AccessToken)
	req.Header.Set("X-Client-ID", s.ClientID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session test failed: %s", resp.Status)
	}

	var out struct {
		Plaintext string `json:"plaintext"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}

	fmt.Println("Successful session testing! Server decrypted:", out.Plaintext)
	return nil
}
