package middleware

import (
	"net/http"
	"strings"

	"github.com/1abobik1/AuthService/pkg/auth/validation"
	"github.com/gin-gonic/gin"
)

func JWTMiddleware(publicKeyPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := validation.ValidateToken(tokenString, publicKeyPath)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}
  