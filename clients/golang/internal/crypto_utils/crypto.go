package crypto_utils

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
	"encoding/pem"
	"fmt"
	"math/big"
)

// DerSig используется для разбора ECDSA-DER подписи
type DerSig struct{ R, S *big.Int }

// Новые SignPayloadECDSA и mustSignPayloadECDSA
func SignPayloadECDSA(priv *ecdsa.PrivateKey, data []byte) (string, error) {
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

// pkcs7Pad добавляет PKCS#7-паддинг до длины кратной blockSize.
func Pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	if padLen == 0 {
		padLen = blockSize
	}
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(data, padding...)
}

func MustSignPayloadECDSA(priv *ecdsa.PrivateKey, data []byte) string {
	sig, err := SignPayloadECDSA(priv, data)
	if err != nil {
		panic(err)
	}
	return sig
}

// BuildEncryptedBlob шифрует входные данные с помощью AES-CBC и дополняет результат HMAC-SHA256
// Возвращает бинарный blob в формате: nonce (16B) || iv (16B) || ciphertext || hmac-tag
func BuildEncryptedBlob(plain, kEnc, kMac []byte) ([]byte, error) {
	// nonce 16 байт
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	// iv 16 байт
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	// AES-CBC + PKCS7
	block, err := aes.NewCipher(kEnc)
	if err != nil {
		return nil, err
	}
	padded := Pkcs7Pad(plain, aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	// HMAC-SHA256(iv || ciphertext)
	mac := hmac.New(sha256.New, kMac)
	mac.Write(iv)
	mac.Write(ciphertext)
	tag := mac.Sum(nil)

	// итоговый буфер: nonce||iv||ciphertext||tag
	buf := bytes.Buffer{}
	buf.Write(nonce)
	buf.Write(iv)
	buf.Write(ciphertext)
	buf.Write(tag)
	return buf.Bytes(), nil
}
