package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
)

type HandshakeReq struct {
	RSAPubClient   string `json:"rsa_pub_client"`
	ECDSAPubClient string `json:"ecdsa_pub_client"`
	Nonce1         string `json:"nonce1"`
	Signature1     string `json:"signature1"`
}

type HandshakeResp struct {
	ClientID       string `json:"client_id"`
	RSAPubServer   string `json:"rsa_pub_server"`
	ECDSAPubServer string `json:"ecdsa_pub_server"`
	Nonce2         string `json:"nonce2"`
	Signature2     string `json:"signature2"`
}

// читаем и возвращаем DER-байты публичного ключа
func loadPub(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("no PEM data in %s", path)
	}
	// если это PUBLIC KEY (PKIX), block.Bytes — DER
	return block.Bytes, nil
}

// читаем приватный ECDSA-ключ
func loadECDSAPriv(path string) (*ecdsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("no PEM in %s", path)
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		// fallback PKCS8
		if ifc, err2 := x509.ParsePKCS8PrivateKey(block.Bytes); err2 == nil {
			if ecd, ok := ifc.(*ecdsa.PrivateKey); ok {
				return ecd, nil
			}
		}
		return nil, err
	}
	return key, nil
}

// fingerprint = hex(SHA256(rsa_pub||ecdsa_pub))
// func fingerprint(rsaPub, ecdsaPub []byte) string {
// 	h := sha256.Sum256(append(rsaPub, ecdsaPub...))
// 	return fmt.Sprintf("%x", h[:])
// }

func main() {
	var (
		rsaPubPath    = flag.String("rsa-pub", "keys/client_rsa.pub", "")
		ecdsaPubPath  = flag.String("ecdsa-pub", "keys/client_ecdsa.pub", "")
		ecdsaPrivPath = flag.String("ecdsa-priv", "keys/client_ecdsa.pem", "")
		serverURL     = flag.String("url", "http://localhost:8080/handshake/init", "")
	)
	flag.Parse()

	rsaPubDER, err := loadPub(*rsaPubPath)
	if err != nil {
		panic(err)
	}
	ecdsaPubDER, err := loadPub(*ecdsaPubPath)
	if err != nil {
		panic(err)
	}
	ecdsaPriv, err := loadECDSAPriv(*ecdsaPrivPath)
	if err != nil {
		panic(err)
	}

	nonce1 := make([]byte, 8)
	if _, err := rand.Read(nonce1); err != nil {
		panic(err)
	}
	//nonce1 := []byte("6Q62iguYP+A=")
	
	// 2) hash1 = SHA256(rsaPub||ecdsaPub||nonce1)
	data1 := append(append(rsaPubDER, ecdsaPubDER...), nonce1...)
	h1 := sha256.Sum256(data1)

	// 3) sign hash1
	r, s, err := ecdsa.Sign(rand.Reader, ecdsaPriv, h1[:])
	if err != nil {
		panic(err)
	}
	derSig, err := asn1.Marshal(der{R: r, S: s})
	if err != nil {
		panic(err)
	}

	// 4) Form request
	reqBody := HandshakeReq{
		RSAPubClient:   base64.StdEncoding.EncodeToString(rsaPubDER),
		ECDSAPubClient: base64.StdEncoding.EncodeToString(ecdsaPubDER),
		Nonce1:         base64.StdEncoding.EncodeToString(nonce1),
		Signature1:     base64.StdEncoding.EncodeToString(derSig),
	}
	js, _ := json.Marshal(reqBody)

	// 5) HTTP POST
	resp, err := http.Post(*serverURL, "application/json", bytes.NewReader(js))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var hr HandshakeResp
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		panic(err)
	}

	fmt.Printf("Handshake response:\n%+v\n", hr)
}

type der struct {
	R *big.Int
	S *big.Int
}
