package handlerToken

import (
	"log"
	"net/http"

	"github.com/1abobik1/AuthService/internal/utils"
	"github.com/gin-gonic/gin"
)

func (h *tokenHandler) handleRefreshToken(refreshToken string) (string, error) {
	expired, claims, err := h.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}

	if expired {
		userID := int(claims["user_id"].(float64))
		userKey := string(claims["user_key"].(string))
		return h.tokenService.UpdateRefreshToken(refreshToken, userID, userKey)
	}

	return refreshToken, nil
}

// TokenUpdate
// @Summary      Обновление access‑токена
// @Description  Берёт refresh‑токен из Cookie и генерирует новые refresh- и access‑токены.
// @Description  Клиент должен вызывать этот endpoint при получении HTTP 401 (реализуйте interceptor на клиенте).
// @Tags         token
// @Accept       json
// @Produce      json
// @Param        Cookie  header    string  true  "Cookie header, например: refresh_token=<token>"
// @Success      200     {object}  map[string]string  "Новый access_token в теле"
// @Failure      401     {string}  string             "Unauthorized — отсутствует или невалидный refresh‑токен"
// @Router       /token/update [post]
func (h *tokenHandler) TokenUpdate(c *gin.Context) {
	const op = "handler.http.token.RefreshToken"

	refreshToken, err := utils.GetRefreshTokenFromCookie(c)
	if err != nil {
		log.Printf("Error getting refresh token: %v, location: %s", err, op)
		c.Status(http.StatusUnauthorized)
		return
	}

	newRefreshToken, err := h.handleRefreshToken(refreshToken)
	if err != nil {
		log.Printf("Error handling refresh token: %v, location: %s", err, op)
		c.Status(http.StatusUnauthorized)
		return
	}

	newAccessToken, err := h.tokenService.UpdateAccessToken(newRefreshToken)
	if err != nil {
		log.Printf("Error updating access token: %v, location: %s", err, op)
		c.Status(http.StatusUnauthorized)
		return
	}

	utils.SetRefreshTokenCookie(c, newRefreshToken)
	c.JSON(http.StatusOK, gin.H{"access_token": newAccessToken})
}
