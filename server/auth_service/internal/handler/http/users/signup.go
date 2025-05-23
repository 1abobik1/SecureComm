package handlerUsers

import (
	"errors"
	"log"
	"net/http"

	"github.com/1abobik1/AuthService/internal/dto"
	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/1abobik1/AuthService/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// SignUp
// @Summary      Регистрация пользователя
// @Description  Создаёт нового пользователя. В зависимости от platform возвращает refresh‑токен в cookie (для web) или в теле ответа (для tg-bot).
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      dto.SignUpDTO  true  "Email, Password и Platform (web или tg-bot)"
// @Success      200   {object}  map[string]string  "Для web: {access_token}, refresh в cookie; для tg-bot: {access_token, refresh_token}"
// @Failure      400   {object}  map[string]string  "error – некорректный запрос или платформа не поддерживается"
// @Failure      409   {object}  map[string]string  "error – пользователь с таким email уже существует"
// @Failure      500   {object}  nil                "Internal Server Error"
// @Router       /user/signup [post]
func (h *userHandler) SignUp(c *gin.Context) {
	const op = "handler.http.users.SignUp"

	var authDTO dto.SignUpDTO

	if err := c.BindJSON(&authDTO); err != nil {
		log.Printf("Error binding JSON: %v location %s\n", err, op)
		c.Status(http.StatusBadRequest)
		return
	}

	validate := validator.New()
	if err := validate.Struct(authDTO); err != nil {
		log.Printf("Error: %s, location: %s", ErrValidation, op)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrValidation})
		return
	}

	if err := utils.ValidatePlatform(authDTO.Platform); err != nil {
		log.Printf("Error: %v, location: %s", err, op)
		c.JSON(http.StatusBadRequest, gin.H{"error": "the error checking platform. available platforms: web, tg-bot"})
		return
	}

	accessToken, refreshToken, err := h.userService.Register(c, authDTO.Email, authDTO.Password, authDTO.Platform)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
			return
		}
		log.Printf("Error Internal logic during user registration. Email: %s, Error: %v \n", authDTO.Email, err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if authDTO.Platform == "tg-bot" {
		// ответ тг боту
		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})

	} else {
		// ответ web
		utils.SetRefreshTokenCookie(c, refreshToken)
		c.JSON(http.StatusOK, gin.H{"access_token": accessToken})
	}
}
