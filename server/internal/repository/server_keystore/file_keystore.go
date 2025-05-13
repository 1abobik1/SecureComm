package server_keystore

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// просто читает и парсит ключи, не генерирует их.
type FileKeyStore struct {
	rsaPriv     *rsa.PrivateKey
	rsaPubPEM   []byte
	ecdsaPriv   *ecdsa.PrivateKey
	ecdsaPubPEM []byte
}


// если хотя бы один файл отсутствует или не парсится — возвращает ошибку.
func NewFileKeyStore(rsaPrivPath, rsaPubPath, ecdsaPrivPath, ecdsaPubPath string) (*FileKeyStore, error) {
	// Проверка и чтение RSA-приватного
	privPEM, err := os.ReadFile(rsaPrivPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read RSA private key %s: %w", rsaPrivPath, err)
	}
	block, _ := pem.Decode(privPEM)
	if block == nil{
		return nil, fmt.Errorf("bad PEM block for RSA private key")
	}

	var rsaPriv *rsa.PrivateKey
	// 1) Попытка PKCS#1
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		rsaPriv = key
	} else {
		// 2) Попытка PKCS#8
		keyIfc, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("cannot parse RSA private key (tried PKCS1: %v; PKCS8: %v)", err, err2)
		}
		var ok bool
		rsaPriv, ok = keyIfc.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS8 key is not RSA: %T", keyIfc)
		}
	}

	// проверка и чтение RSA-публичного
	rsaPubPEM, err := os.ReadFile(rsaPubPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read RSA public key %s: %w", rsaPubPath, err)
	}

	// проверка и чтение ECDSA-приватного
	ecdsaPrivPEM, err := os.ReadFile(ecdsaPrivPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read ECDSA private key %s: %w", ecdsaPrivPath, err)
	}
	block, _ = pem.Decode(ecdsaPrivPEM)
	if block == nil {
		return nil, fmt.Errorf("bad PEM block for ECDSA private key")
	}

	var ecdsaPriv *ecdsa.PrivateKey
	// попытка парсинга SEC1
	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		ecdsaPriv = key
	} else {
		// или PKCS#8
		keyIfc, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("invalid ECDSA private key format: %v / %v", err, err2)
		}
		var ok bool
		ecdsaPriv, ok = keyIfc.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("parsed key is not ECDSA")
		}
	}

	// проверка и чтение ECDSA-публичного
	ecdsaPubPEM, err := os.ReadFile(ecdsaPubPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read ECDSA public key %s: %w", ecdsaPubPath, err)
	}

	return &FileKeyStore{
		rsaPriv:     rsaPriv,
		rsaPubPEM:   rsaPubPEM,
		ecdsaPriv:   ecdsaPriv,
		ecdsaPubPEM: ecdsaPubPEM,
	}, nil
}

// возвращает уже считанные ключи
func (ks *FileKeyStore) GetServerKeys() (rsaPriv *rsa.PrivateKey, rsaPub []byte, ecdsaPriv *ecdsa.PrivateKey, ecdsaPub []byte) {
	return ks.rsaPriv, ks.rsaPubPEM, ks.ecdsaPriv, ks.ecdsaPubPEM
}
