package utils

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Структура данных для хранения пользовательских claims
type customClaims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// Создание Access Token
func CreateAccessToken(userID int, duration time.Duration, privateKeyPath string) (string, error) {
	privateKey, err := getgPrivateKey(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("error to get private key in file: %s", privateKeyPath)
	}

	// Настраиваем claims для access токена
	claims := customClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)), // Устанавливаем срок действия токена
			IssuedAt:  jwt.NewNumericDate(time.Now()),               // Время выпуска токена
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Подписываем токен секретным ключом
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", err // Возвращаем ошибку, если подпись не удалась
	}

	return tokenString, nil // Возвращаем готовый токен
}

// Создание Refresh Token
func CreateRefreshToken(userID int, duration time.Duration, privateKeyPath string) (string, error) {

	privateKey, err := getgPrivateKey(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("error to get private key in file: %s", privateKeyPath)
	}
	// Настраиваем claims для refresh токена
	claims := customClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)), // Срок действия токена
			IssuedAt:  jwt.NewNumericDate(time.Now()),               // Время выпуска токена
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Подписываем токен секретным ключом
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", err // Возвращаем ошибку, если подпись не удалась
	}

	return tokenString, nil // Возвращаем готовый токен
}

func getgPrivateKey(file string) (*rsa.PrivateKey, error) {
	// Читаем приватный ключ из файла
	privateKeyData, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}
