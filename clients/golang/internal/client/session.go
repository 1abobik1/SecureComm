package client

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// Session хранит состояние после успешного finalize
type Session struct {
	ClientID string
	Ks       []byte // 32‑байтовый секрет
	KEnc     []byte // ключ для AES-256-CBC
	KMac     []byte // ключ для HMAC-SHA256
	TestURL  string // url для /session/test
}

// NewSession инициализирует Session из ks и clientID
func NewSession(clientID string, ks []byte, testURL string) *Session {
	// деривация ключей из ks
	mac := hmac.New(sha256.New, ks)
	mac.Write([]byte("mac"))
	kmac := mac.Sum(nil)

	enc := hmac.New(sha256.New, ks)
	enc.Write([]byte("enc"))
	kenc := enc.Sum(nil)

	return &Session{
		ClientID: clientID,
		Ks:       ks,
		KEnc:     kenc,
		KMac:     kmac,
		TestURL:  testURL,
	}
}

// DoSessionTest шлёт зашифрованное сообщение и печатает расшифрованный ответ
func (s *Session) DoSessionTest(plaintext string) error {
	// IV
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}
	block, err := aes.NewCipher(s.KEnc)
	if err != nil {
		return err
	}
	padded := pkcs7Pad([]byte(plaintext))
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	// HMAC
	mac := hmac.New(sha256.New, s.KMac)
	mac.Write(iv)
	mac.Write(ciphertext)
	tag := mac.Sum(nil)

	// упаковать и в Base64
	pkg := append(iv, ciphertext...)
	pkg = append(pkg, tag...)
	b64msg := base64.StdEncoding.EncodeToString(pkg)

	// отправка
	reqBody := map[string]string{"encrypted_message": b64msg}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.TestURL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
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

	fmt.Println("Successful session testing!")
	fmt.Println("Server decrypted:", out.Plaintext)

	return nil
}
