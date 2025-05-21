package client

import (
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
)

type DerSig struct{ R, S *big.Int }

func SignPayloadECDSA(priv *ecdsa.PrivateKey, data []byte) (string, error) {
	//fmt.Println("\n\n\n",priv.X, "\n\n\n")

	h := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, priv, h[:])
	if err != nil {
		return "", err
	}
	der, err := asn1.Marshal(DerSig{R: r, S: s})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(der), nil
}

func GenerateRandBytes(size int) (string, []byte, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", nil, err
	}
	return base64.StdEncoding.EncodeToString(buf), buf, nil
}

func EncryptRSA(pub *rsa.PublicKey, plaintext []byte) (string, error) {
	cipher, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, plaintext, nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(cipher), nil
}

func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func LoadRSAPubDER(pemBytes []byte) ([]byte, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	return block.Bytes, nil
}

func ParseECDSAPriv(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}
	if ifc, err2 := x509.ParsePKCS8PrivateKey(block.Bytes); err2 == nil {
		if k, ok := ifc.(*ecdsa.PrivateKey); ok {
			return k, nil
		}
	}
	return nil, fmt.Errorf("unable to parse ECDSA private key")
}

func pkcs7Pad(data []byte) []byte {
	pad := aes.BlockSize - len(data)%aes.BlockSize
	for i := 0; i < pad; i++ {
		data = append(data, byte(pad))
	}
	return data
}
