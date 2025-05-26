package client

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"example_client/internal/crypto_utils"
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
	nonce1b64, nonce1, err := crypto_utils.GenerateRandBytes(8)
	if err != nil {
		return nil, err
	}
	toSign1 := append(append(rsaPubClientDER, ecdsaPubClientDER...), nonce1...)
	sig1b64, err := crypto_utils.SignPayloadECDSA(ecdsaPriv, toSign1)
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

// Finalize Handshake (с Authorization)
func DoFinalizeAPI(url, sessionTestURL string, initResp *dto.HandshakeResp, ecdsaPriv *ecdsa.PrivateKey, accessToken string) (*Session, error) {
	// парсим RSA-публичный ключ сервера
	rawRSAPubDER, err := base64.StdEncoding.DecodeString(initResp.RSAPubServer)
	if err != nil {
		return nil, err
	}
	piRSA, err := x509.ParsePKIXPublicKey(rawRSAPubDER)
	if err != nil {
		return nil, fmt.Errorf("finalize: cannot parse server RSA pub")
	}
	rsaPubSrv := piRSA.(*rsa.PublicKey)

	// декодируем nonce2
	nonce2, err := base64.StdEncoding.DecodeString(initResp.Nonce2)
	if err != nil {
		return nil, err
	}

	// генерируем ks (32 байта) и nonce3 (8 байт)
	ks := make([]byte, 32)
	if _, err := rand.Read(ks); err != nil {
		return nil, err
	}
	_, nonce3, err := crypto_utils.GenerateRandBytes(8)
	if err != nil {
		return nil, err
	}

	// собираем payload = ks || nonce3 || nonce2
	payload := append(append(ks, nonce3...), nonce2...)
	// подписываем его ECDSA
	sig3b64, err := crypto_utils.SignPayloadECDSA(ecdsaPriv, payload)
	if err != nil {
		return nil, err
	}

	// шифруем payload RSA-OAEP
	payloadCipherB64, err := crypto_utils.EncryptRSA(rsaPubSrv, payload)
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

	// проверяем Signature4
	sig4DER, err := base64.StdEncoding.DecodeString(fr.Signature4)
	if err != nil {
		return nil, err
	}
	var sig4 crypto_utils.DerSig
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

	// проверяем подпись4: sha256( ks || nonce3 || nonce2 )
	h4 := sha256.Sum256(append(append(ks, nonce3...), nonce2...))
	if !ecdsa.Verify(serverECDSAPub, h4[:], sig4.R, sig4.S) {
		return nil, fmt.Errorf("finalize: bad server signature4")
	}

	// Инициализируем Session
	return NewSession(initResp.ClientID, ecdsaPriv, ks, sessionTestURL, accessToken), nil
}
