package middleware

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
)

var ErrTokenExpired = errors.New("token is expired")
var ErrTokenInvalid = errors.New("invalid token")

func JWTMiddleware(publicKeyPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logrus.Error("Error auth header: invalid token format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := ValidateToken(tokenString, publicKeyPath)
		if err != nil {
			logrus.Errorf("Error ValidateToken: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

func ValidateToken(tokenString, publicKeyPath string) (jwt.MapClaims, error) {
	publicKey, err := getPublicKey(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading public key from file: %s", publicKeyPath)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, ErrTokenInvalid
		}
		return publicKey, nil
	})

	if err != nil {
		var validationError *jwt.ValidationError
		if errors.As(err, &validationError) && (validationError.Errors&jwt.ValidationErrorExpired != 0) && token != nil {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				return claims, ErrTokenExpired
			}
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

func getPublicKey(file string) (*rsa.PublicKey, error) {
	publicKeyData, err := os.ReadFile(file)
	if err != nil {
		logrus.Error("Error to parse jwt pub key")
		return nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		logrus.Error("Error to parse ParseRSAPublicKeyFromPEM func")
		return nil, err
	}

	return publicKey, nil
}