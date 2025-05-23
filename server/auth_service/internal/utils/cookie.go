package utils

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func GetRefreshTokenFromCookie(c *gin.Context) (string, error) {
	token, err := c.Cookie("refresh_token")
	if err != nil || len(token) == 0 {
		return "", fmt.Errorf("refresh token not found or empty")
	}
	return token, nil
}

func SetRefreshTokenCookie(c *gin.Context, refreshToken string) {
	c.SetCookie(
		"refresh_token",
		refreshToken,
		30*24*60*60, // 30 дней
		"/",
		"",
		false, // true для HTTPS
		true,  // доступен только через HTTP
	)
}
