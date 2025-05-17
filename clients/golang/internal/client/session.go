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
	"fmt"
	"net/http"
	"time"
)

// Session хранит состояние после успешного finalize
type Session struct {
	ClientID  string
	ECDSAPriv *ecdsa.PrivateKey
	Ks        []byte // 32‑байтовый секрет
	KEnc      []byte // ключ для AES-256-CBC
	KMac      []byte // ключ для HMAC-SHA256
	TestURL   string // url для /session/test
}

// NewSession инициализирует Session из ks и clientID
func NewSession(clientID string, ecdsaPriv *ecdsa.PrivateKey, ks []byte, testURL string) *Session {
	// деривация ключей из ks
	mac := hmac.New(sha256.New, ks)
	mac.Write([]byte("mac"))
	kmac := mac.Sum(nil)

	enc := hmac.New(sha256.New, ks)
	enc.Write([]byte("enc"))
	kenc := enc.Sum(nil)

	return &Session{
		ClientID:  clientID,
		ECDSAPriv: ecdsaPriv,
		Ks:        ks,
		KEnc:      kenc,
		KMac:      kmac,
		TestURL:   testURL,
	}
}

// DoSessionTest шлёт зашифрованное сообщение и печатает расшифрованный ответ
func (s *Session) DoSessionTest(plaintext string) error {
	// собираем metadata - это timestamp + nonce
	ts := time.Now().UnixMilli()
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(ts))

	// генерация nonce
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	// nonce := []byte("sAQG5pFgEho=sAQG")

	// формируем blob = metadata || userData(plaintext)
	totalLen := len(timestamp) + len(nonce) + len(plaintext)
	blob := make([]byte, 0, totalLen)
	blob = append(blob, timestamp...)
	blob = append(blob, nonce...)
	blob = append(blob, []byte(plaintext)...)

	// генерация iv
	iv := make([]byte, aes.BlockSize)
	rand.Read(iv)
	// создание aes блока
	block, _ := aes.NewCipher(s.KEnc)
	// PKCS#7 padding
	padded := pkcs7Pad(blob)

	// шифрование CBC
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	// вычисление HMAC‑SHA256
	mac := hmac.New(sha256.New, s.KMac)
	mac.Write(iv)
	mac.Write(ciphertext)
	tag := mac.Sum(nil)

	// формирование конечного пакета
	totalLenPKG := len(iv) + len(ciphertext) + len(tag)
	pkg := make([]byte, 0, totalLenPKG)
	pkg = append(pkg, iv...)
	pkg = append(pkg, ciphertext...)
	pkg = append(pkg, tag...)
	b64 := base64.StdEncoding.EncodeToString(pkg)

	// подписываем приватным ключом клиента
	signatureB64, err := SignPayloadECDSA(s.ECDSAPriv, pkg)
	if err != nil {
		panic(err)
	}

	// отправка
	reqBody := map[string]string{
		"encrypted_message": b64,
		"client_signature":  signatureB64,
	}
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
	//fmt.Println("Server decrypted:", out.Plaintext)

	return nil
}
