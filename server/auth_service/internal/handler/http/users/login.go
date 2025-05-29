package handlerUsers

import (
	"errors"
	"log"
	"net/http"

	"github.com/1abobik1/AuthService/internal/dto"
	"github.com/1abobik1/AuthService/internal/middleware"
	serviceUsers "github.com/1abobik1/AuthService/internal/service/users"
	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/1abobik1/AuthService/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Login
// @Summary      Аутентификация пользователя
// @Description  Логин по email и паролю.
// @Description  В зависимости от поля `platform` в запросе возвращаются разные данные:
// @Description  Для platform="tg-bot":
// @Description  access_token
// @Description  refresh_token
// @Description  k_enc(Base64)
// @Description  k_mac(Base64)
// @Description
// @Description  Для platform="web":
// @Description  access_token
// @Description  ks(JSON-объект с полями `k_enc_iv`, `k_enc_data`, `k_mac_iv`, `k_mac_data`)
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LogInDTO  true  "Email, Password и Platform (web или tg-bot)"
// @Success      200   {object}  map[string]interface{}  "Поля ответа зависят от платформы (см. описание выше)"
// @Failure      400   {object}  map[string]string       "Bad request или неверный формат platform"
// @Failure      403   {object}  map[string]string       "incorrect password or email"
// @Failure      404   {object}  map[string]string       "user not found"
// @Failure      500   {string}  string                  "Internal Server Error"
// @Router       /user/login [post]
func (h *userHandler) Login(c *gin.Context) {
	const op = "handler.http.users.Login"

	var authDTO dto.LogInDTO

	if err := c.BindJSON(&authDTO); err != nil {
		log.Printf("Error binding JSON: %v, location %s", err, op)
		c.Set("failed_registration", true)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	validate := validator.New()
	if err := validate.Struct(authDTO); err != nil {
		log.Printf("Error: %s, location: %s", ErrValidation, op)
		c.Set("failed_registration", true)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrValidation})
		return
	}

	if err := utils.ValidatePlatform(authDTO.Platform); err != nil {
		log.Printf("Error: %v, location: %s", err, op)
		c.Set("failed_registration", true)
		c.JSON(http.StatusBadRequest, gin.H{"error": "the error checking platform. available platforms: web, tg-bot"})
		return
	}

	accessToken, refreshToken, err := h.userService.Login(c, authDTO.Email, authDTO.Password, authDTO.Platform)
	if err != nil {
		if errors.Is(err, serviceUsers.ErrInvalidCredentials) {
			log.Printf("Error: %v", err)
			c.Set("failed_registration", true)
			c.JSON(http.StatusForbidden, gin.H{"error": "incorrect password or email"})
			return
		}
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("Error: %v", err)
			c.Set("failed_registration", true)
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		log.Printf("Error: %v", err)
		c.Set("failed_registration", true)
		c.Status(http.StatusInternalServerError)
		return
	}

	ip := c.ClientIP()
	middleware.FailedAttemptsCache.Delete("fail_" + ip)

	if authDTO.Platform == "tg-bot" {
		// если клиент зашел с тг бота
		kEncB64, kMacB64, err := h.tgClient.GetClientKS(c, accessToken)
		if err != nil {
			c.Set("failed_registration", true)
			c.JSON(http.StatusBadRequest, gin.H{"error": "error in receiving session key and ecdsa"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"k_enc":         kEncB64,
			"k_mac":         kMacB64,
		})
	} else {
		// если клиент зашел с web сайта, ks передается в зашифрованной ввиде паролем пользователя
		ksB64, err := h.webClient.GetClientKS(c, authDTO.Password, accessToken)
		if err != nil {
			c.Set("failed_registration", true)
			c.JSON(http.StatusBadRequest, gin.H{"error": "error in receiving session key and ecdsa"})
			return
		}

		utils.SetRefreshTokenCookie(c, refreshToken)

		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"ks":           ksB64,
		})
	}
}
