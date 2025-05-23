package checker

import (
	"fmt"
	"os"
)

// EnsureKeys проверяет наличие ключей(публичный и приватный rsa, а также публичный и приватный ecdsa) в dir и генерирует новые при отсутствии.
func CheckKeys(privRSA, pubRSA, privECDSA, pubECDSA string) error {
	// RSA
	if !fileExists(privRSA) || !fileExists(pubRSA) {
		return fmt.Errorf("the RSA key does not exist")
	}

	// ECDSA
	if !fileExists(privECDSA) || !fileExists(pubECDSA) {
		return fmt.Errorf("the ECDSA key does not exist")
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// func genRSA(privPath, pubPath string) error {
// 	key, err := rsa.GenerateKey(rand.Reader, 3072)
// 	if err != nil {
// 		return err
// 	}
// 	// Сериализуем приватный
// 	privBytes := x509.MarshalPKCS1PrivateKey(key)
// 	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
// 	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
// 		return err
// 	}
// 	// Публичный
// 	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
// 	if err != nil {
// 		return err
// 	}
// 	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
// 	return os.WriteFile(pubPath, pubPEM, 0644)
// }

// func genECDSA(privPath, pubPath string) error {
// 	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
// 	if err != nil {
// 		return err
// 	}
// 	privBytes, err := x509.MarshalECPrivateKey(key)
// 	if err != nil {
// 		return err
// 	}
// 	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
// 	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
// 		return err
// 	}
// 	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
// 	if err != nil {
// 		return err
// 	}
// 	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
// 	return os.WriteFile(pubPath, pubPEM, 0644)
// }
