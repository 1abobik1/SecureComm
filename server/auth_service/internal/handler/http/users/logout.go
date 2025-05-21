package handlerUsers

import (
	"net/http"

	"log"

	"github.com/1abobik1/AuthService/internal/utils"
	"github.com/gin-gonic/gin"
)

type LogoutRequest struct {
	Platform     string `json:"platform"     binding:"required,oneof=web tg-bot"`
	RefreshToken string `json:"refresh_token"` // для tg-bot
}

// Logout
// @Summary      Выход (logout)
// @Description  Отзывает refresh-токен. Для web берёт токен из cookie, для tg-bot — из JSON body. Для веба не надо передавать refresh_token в json body
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      handlerUsers.LogoutRequest  false  "{" +
//                                                              "\"platform\":\"web|tg-bot\", " +
//                                                              "\"refresh_token\":\"...\" (для tg-bot)" +
//                                                              "}"
// @Success      200   {string}  string                     "OK"
// @Failure      400   {object}  map[string]string          "error — некорректный запрос или платформа не поддерживается"
// @Failure      401   {string}  string                     "Unauthorized — отсутствует или невалидный токен"
// @Failure      500   {string}  string                     "Internal Server Error"
// @Router       /users/logout [post]
func (h *userHandler) Logout(c *gin.Context) {
	const op = "handler.http.users.Logout"

	var req LogoutRequest
	// Попробуем распарсить JSON — если это телеграм‑бот, там будет { platform:"tg-bot", refresh_token: "..." }
	if err := c.BindJSON(&req); err != nil {
		// Не бот? Считаем, что web: platform="web"
		req.Platform = "web"
	}

	var refreshToken string
	var err error

	switch req.Platform {
	case "tg-bot":
		refreshToken = req.RefreshToken
		if refreshToken == "" {
			log.Printf("Logout: missing refresh_token in body, location %s\n", op)
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing refresh_token"})
			return
		}

	case "web":
		refreshToken, err = utils.GetRefreshTokenFromCookie(c)
		if err != nil {
			log.Printf("Error getting refresh token from cookie: %v, location: %s\n", err, op)
			c.Status(http.StatusUnauthorized)
			return
		}

	default:
		// Защита от будущих расширений
		log.Printf("Unsupported platform %q, location: %s\n", req.Platform, op)
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform not supported"})
		return
	}

	// Отзываем токен в сервисе
	if err := h.userService.RevokeRefreshToken(c, refreshToken); err != nil {
		log.Printf("Error revoking refresh token: %v, location: %s\n", err, op)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Для web сбрасываем куку
	if req.Platform == "web" {
		c.SetCookie(
			"refresh_token",
			"",
			-1,
			"/",
			"",
			false,
			true,
		)
	}

	// В обоих случаях просто 200 OK
	c.Status(http.StatusOK)
}
