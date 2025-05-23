package client

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"example_client/internal/dto"
)

func PostJSON(url string, payload interface{}, headers map[string]string) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return http.DefaultClient.Do(req)
}

// ------------------------------
// 1) SignUp: вызываем /user/signup и получаем access_token, refresh_token
// ------------------------------
func DoSignUpAPI(signupURL, email, password, platform string) (accessToken, refreshToken string, err error) {
	reqBody := dto.SignUpDTO{
		Email:    email,
		Password: password,
		Platform: platform,
	}
	resp, err := PostJSON(signupURL, reqBody, nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("signup failed: status %d, body %q", resp.StatusCode, string(b))
	}

	var out map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", fmt.Errorf("signup: invalid JSON: %w", err)
	}

	accessToken = out["access_token"]
	refreshToken = out["refresh_token"]
	return accessToken, refreshToken, nil
}

// ------------------------------
// 2) Init Handshake с заголовком Authorization
// ------------------------------
func DoInitAPI(
	url string,
	rsaPubClientDER, ecdsaPubClientDER []byte,
	ecdsaPriv *ecdsa.PrivateKey,
	accessToken string,
) (*dto.HandshakeResp, error) {
	// Генерация nonce1 и подпись clientRSA||clientECDSA||nonce1
	nonce1b64, nonce1, err := GenerateRandBytes(8)
	if err != nil {
		return nil, err
	}
	toSign1 := append(append(rsaPubClientDER, ecdsaPubClientDER...), nonce1...)
	sig1b64, err := SignPayloadECDSA(ecdsaPriv, toSign1)
	if err != nil {
		return nil, err
	}

	// Запрос
	reqBody := dto.HandshakeReq{
		RSAPubClient:   base64.StdEncoding.EncodeToString(rsaPubClientDER),
		ECDSAPubClient: base64.StdEncoding.EncodeToString(ecdsaPubClientDER),
		Nonce1:         nonce1b64,
		Signature1:     sig1b64,
	}
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}
	resp, err := PostJSON(url, reqBody, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("handshake/init failed: status %d, body %q", resp.StatusCode, string(b))
	}

	var hr dto.HandshakeResp
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, fmt.Errorf("handshake/init: invalid JSON: %w", err)
	}
	return &hr, nil
}

// ------------------------------
// 3) Finalize Handshake (с Authorization)
// ------------------------------
func DoFinalizeAPI(
	url, sessionTestURL string,
	initResp *dto.HandshakeResp,
	ecdsaPriv *ecdsa.PrivateKey,
	accessToken string,
) (*Session, error) {
	// Парсим RSA-публичный ключ сервера
	rawRSAPubDER, err := base64.StdEncoding.DecodeString(initResp.RSAPubServer)
	if err != nil {
		return nil, err
	}
	piRSA, err := x509.ParsePKIXPublicKey(rawRSAPubDER)
	if err != nil {
		return nil, fmt.Errorf("finalize: cannot parse server RSA pub")
	}
	rsaPubSrv := piRSA.(*rsa.PublicKey)

	// Декодируем nonce2
	nonce2, err := base64.StdEncoding.DecodeString(initResp.Nonce2)
	if err != nil {
		return nil, err
	}

	// Генерируем ks (32 байта) и nonce3 (8 байт)
	ks := make([]byte, 32)
	if _, err := rand.Read(ks); err != nil {
		return nil, err
	}
	_, nonce3, err := GenerateRandBytes(8)
	if err != nil {
		return nil, err
	}

	// Собираем payload = ks || nonce3 || nonce2
	payload := append(append(ks, nonce3...), nonce2...)
	// Подписываем его ECDSA
	sig3b64, err := SignPayloadECDSA(ecdsaPriv, payload)
	if err != nil {
		return nil, err
	}

	// Шифруем payload RSA-OAEP
	payloadCipherB64, err := EncryptRSA(rsaPubSrv, payload)
	if err != nil {
		return nil, err
	}

	// Finalize-запрос
	reqBody := dto.FinalizeReq{
		Encrypted:  payloadCipherB64,
		Signature3: sig3b64,
	}
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"X-Client-ID":   initResp.ClientID,
	}
	resp, err := PostJSON(url, reqBody, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("handshake/finalize failed: status %d, body %q", resp.StatusCode, string(b))
	}

	var fr dto.FinalizeResp
	if err := json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return nil, fmt.Errorf("handshake/finalize: invalid JSON: %w", err)
	}

	// Проверяем Signature4
	sig4DER, err := base64.StdEncoding.DecodeString(fr.Signature4)
	if err != nil {
		return nil, err
	}
	var sig4 DerSig
	if _, err := asn1.Unmarshal(sig4DER, &sig4); err != nil {
		return nil, fmt.Errorf("finalize: invalid server signature4 DER")
	}
	rawECDSAPubDER, err := base64.StdEncoding.DecodeString(initResp.ECDSAPubServer)
	if err != nil {
		return nil, err
	}
	piECDSA, err := x509.ParsePKIXPublicKey(rawECDSAPubDER)
	if err != nil {
		return nil, fmt.Errorf("finalize: cannot parse server ECDSA pub")
	}
	serverECDSAPub := piECDSA.(*ecdsa.PublicKey)

	// Проверяем подпись4: sha256( ks || nonce3 || nonce2 )
	h4 := sha256.Sum256(append(append(ks, nonce3...), nonce2...))
	if !ecdsa.Verify(serverECDSAPub, h4[:], sig4.R, sig4.S) {
		return nil, fmt.Errorf("finalize: bad server signature4")
	}

	// Инициализируем Session
	return NewSession(initResp.ClientID, ecdsaPriv, ks, sessionTestURL, accessToken), nil
}

// ------------------------------
// 4) Session Test (с Authorization)
// ------------------------------
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

	// Собираем blob = timestamp||nonce||plaintext
	blob := append(append(timestamp, nonce...), []byte(plaintext)...)

	// iv (16 байт)
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}

	block, _ := aes.NewCipher(s.KEnc)
	padded := Pkcs7Pad(blob, aes.BlockSize)
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
		"client_signature":  mustSignPayloadECDSA(s.ECDSAPriv, pkg),
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
