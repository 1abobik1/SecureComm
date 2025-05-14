package client

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

func DoInitAPI(url string, rsaPubDER, ecdsaPubDER []byte, ecdsaPriv *ecdsa.PrivateKey) *dto.HandshakeResp {
	// генерим nonce1 и подписываем его вместе с публичными ключами клиента
	nonce1b64, nonce1, err := GenerateRandBytes(8)
	if err != nil {
		panic(err)
	}
	toSign1 := append(append(rsaPubDER, ecdsaPubDER...), nonce1...)
	sig1b64, err := SignPayloadECDSA(ecdsaPriv, toSign1)
	if err != nil {
		panic(err)
	}

	// отправляем Init-запрос
	reqBody := dto.HandshakeReq{
		RSAPubClient:   base64.StdEncoding.EncodeToString(rsaPubDER),
		ECDSAPubClient: base64.StdEncoding.EncodeToString(ecdsaPubDER),
		Nonce1:         nonce1b64,
		Signature1:     sig1b64,
	}
	resp, err := PostJSON(url, reqBody, nil)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		panic(fmt.Errorf("handshake/init failed: status %d, body: %q", resp.StatusCode, string(b)))
	}

	var hr dto.HandshakeResp
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		panic(fmt.Errorf("handshake/init: invalid JSON: %w", err))
	}

	// декодируем nonce2 и sig2
	nonce2, err := base64.StdEncoding.DecodeString(hr.Nonce2)
	if err != nil {
		panic(err)
	}
	sig2DER, err := base64.StdEncoding.DecodeString(hr.Signature2)
	if err != nil {
		panic(err)
	}
	var sig2 DerSig
	if _, err := asn1.Unmarshal(sig2DER, &sig2); err != nil {
		panic("handshake/init: invalid server signature DER")
	}

	// декодируем публичные ключи сервера
	rawECDSAPubDER, err := base64.StdEncoding.DecodeString(hr.ECDSAPubServer)
	if err != nil {
		panic(err)
	}
	piECDSA, err := x509.ParsePKIXPublicKey(rawECDSAPubDER)
	if err != nil {
		panic("handshake/init: cannot parse server ECDSA pub")
	}
	serverECDSAPub := piECDSA.(*ecdsa.PublicKey)

	rawRSAPubDER, err := base64.StdEncoding.DecodeString(hr.RSAPubServer)
	if err != nil {
		panic(err)
	}
	piRSA, err := x509.ParsePKIXPublicKey(rawRSAPubDER)
	if err != nil {
		panic("handshake/init: cannot parse server RSA pub")
	}
	_ = piRSA.(*rsa.PublicKey)

	// верифицируем sig2
	clientID := hr.ClientID
	data2 := append(append(append(append(rawRSAPubDER, rawECDSAPubDER...), nonce2...), nonce1...), clientID...)
	h2 := sha256.Sum256(data2)
	if !ecdsa.Verify(serverECDSAPub, h2[:], sig2.R, sig2.S) {
		panic("handshake/init: server signature verification failed")
	}

	return &hr
}

func DoFinalizeAPI(url string, initResp *dto.HandshakeResp, ecdsaPriv *ecdsa.PrivateKey) dto.FinalizeResp {
	// декодируем RSA публичный ключ сервера
	rsaPubDER, err := base64.StdEncoding.DecodeString(initResp.RSAPubServer)
	if err != nil {
		panic(err)
	}
	piRSA, err := x509.ParsePKIXPublicKey(rsaPubDER)
	if err != nil {
		panic("finalize: cannot parse RSA server key")
	}
	rsaPubSrv := piRSA.(*rsa.PublicKey)

	// декодируем nonce2
	nonce2, err := base64.StdEncoding.DecodeString(initResp.Nonce2)
	if err != nil {
		panic(err)
	}

	// генерируем ks и nonce3
	_, ks, err := GenerateRandBytes(32)
	if err != nil {
		panic(err)
	}
	_, nonce3, err := GenerateRandBytes(8)
	if err != nil {
		panic(err)
	}

	// подписываем payload = ks || nonce3 || nonce2
	payload := append(append(ks, nonce3...), nonce2...)
	sig3b64, err := SignPayloadECDSA(ecdsaPriv, payload)
	if err != nil {
		panic(err)
	}
	// декодируем base64 -> raw DER
	sig3DER, err := base64.StdEncoding.DecodeString(sig3b64)
	if err != nil {
		panic(err)
	}

	//  шифруем payload || sig3DER
	toEncrypt := append(payload, sig3DER...)
	cipherB64, err := EncryptRSA(rsaPubSrv, toEncrypt)
	if err != nil {
		panic(err)
	}

	// отправляем Finalize-запрос
	reqBody := dto.FinalizeReq{Encrypted: cipherB64}
	headers := map[string]string{"X-Client-ID": initResp.ClientID}
	resp, err := PostJSON(url, reqBody, headers)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Finalize status:", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		panic(fmt.Errorf("handshake/finalize failed: status %d, body: %q", resp.StatusCode, string(b)))
	}

	var fr dto.FinalizeResp
	if err := json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		panic(fmt.Errorf("handshake/finalize: invalid JSON: %w", err))
	}

	// декодируем и проверяем подпись4
	sig4DER, err := base64.StdEncoding.DecodeString(fr.Signature4)
	if err != nil {
		panic(err)
	}
	var sig4 DerSig
	if _, err := asn1.Unmarshal(sig4DER, &sig4); err != nil {
		panic("finalize: invalid server signature4 DER")
	}

	// парсим ECDSA публичный ключ сервера
	rawECDSAPubDER, err := base64.StdEncoding.DecodeString(initResp.ECDSAPubServer)
	if err != nil {
		panic(err)
	}
	piECDSA, err := x509.ParsePKIXPublicKey(rawECDSAPubDER)
	if err != nil {
		panic("finalize: cannot parse ECDSA server key")
	}
	serverECDSAPub := piECDSA.(*ecdsa.PublicKey)

	//  проверяем подпись4: SHA256(ks || nonce3 || nonce2)
	h4 := sha256.Sum256(append(append(ks, nonce3...), nonce2...))
	if !ecdsa.Verify(serverECDSAPub, h4[:], sig4.R, sig4.S) {
		panic("finalize: bad server signature4")
	}

	fmt.Println("Finalize OK, server signature verified")
	return fr
}