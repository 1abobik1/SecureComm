package client

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"example_client/internal/dto"
	"fmt"
	"net/http"
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

func DoInitAPI(url string, rsaPubDER, ecdsaPubDER []byte, ecdsaPriv interface{}) *dto.HandshakeResp {
	// generate nonce and signature
	nonceB64, nonce, err := GenerateNonce(8)
	if err != nil {
		panic(err)
	}
	data := append(append(rsaPubDER, ecdsaPubDER...), nonce...)
	sig, err := SignPayloadECDSA(ecdsaPriv.(*ecdsa.PrivateKey), data)
	if err != nil {
		panic(err)
	}
	req := dto.HandshakeReq{
		RSAPubClient:   base64.StdEncoding.EncodeToString(rsaPubDER),
		ECDSAPubClient: base64.StdEncoding.EncodeToString(ecdsaPubDER),
		Nonce1:         nonceB64,
		Signature1:     sig,
	}
	resp, err := PostJSON(url, req, nil)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var hr dto.HandshakeResp
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		panic(err)
	}
	return &hr
}

func DoFinalizeAPI(url string, initResp *dto.HandshakeResp, ecdsaPriv interface{}) {
	// decode server RSA pub
	srvRSADer, _ := base64.StdEncoding.DecodeString(initResp.RSAPubServer)
	pubIfc, _ := x509.ParsePKIXPublicKey(srvRSADer)
	rsaPubSrv := pubIfc.(*rsa.PublicKey)
	// decode nonce2
	nonce2, _ := base64.StdEncoding.DecodeString(initResp.Nonce2)
	// generate ks and nonce3
	_, ks, _ := GenerateNonce(32)
	_, nonce3, _ := GenerateNonce(8)
	payload := append(append(ks, nonce3...), nonce2...)
	sig3, err := SignPayloadECDSA(ecdsaPriv.(*ecdsa.PrivateKey), payload)
	if err != nil {
		panic(err)
	}
	toEncrypt := append(payload, []byte(sig3)...)
	cipherB64, err := EncryptRSA(rsaPubSrv, toEncrypt)
	if err != nil {
		panic(err)
	}
	req := dto.FinalizeReq{Encrypted: cipherB64}
	headers := map[string]string{"X-Client-ID": initResp.ClientID}
	finResp, err := PostJSON(url, req, headers)
	if err != nil {
		panic(err)
	}
	defer finResp.Body.Close()
	fmt.Println("Finalize status:", finResp.StatusCode)
}
