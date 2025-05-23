package fileloader

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// loadDERPub читает PEM-файл и возвращает DER-байты
func LoadDERPub(path string) ([]byte, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM data in %s", path)
	}

	return block.Bytes, nil
}

// loadECDSAPriv читает ECDSA-ключ из PEM
func LoadECDSAPriv(path string) (*ecdsa.PrivateKey, error) {
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
		if ifc, err2 := x509.ParsePKCS8PrivateKey(block.Bytes); err2 == nil {
			if ecd, ok := ifc.(*ecdsa.PrivateKey); ok {
				return ecd, nil
			}
		}
		return nil, err
	}
	return key, nil
}
