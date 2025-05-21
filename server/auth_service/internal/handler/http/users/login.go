package handlerUsers

import (
	"errors"
	"log"
	"net/http"

	"github.com/1abobik1/AuthService/internal/dto"
	serviceUsers "github.com/1abobik1/AuthService/internal/service/users"
	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/1abobik1/AuthService/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Login
// @Summary      Аутентификация пользователя
// @Description  Логин по email и паролю.  
// @Description  Если platform="tg-bot", возвращаются в JSON:  
// @Description    - access_token  
// @Description    - refresh_token  
// @Description    - ecdsa_priv_client (Base64)  // TODO(СДЕЛАЮ ПОПОЗЖЕ(ПОКА ПРОСТО ЗАТЫЧКИ)): во внешнем API передавать в «голом» виде  
// @Description    - ks (Base64)                // TODO(СДЕЛАЮ ПОПОЗЖЕ(ПОКА ПРОСТО ЗАТЫЧКИ)): во внешнем API передавать в «голом» виде  
// @Description  Если platform="web", возвращаются:  
// @Description    - access_token в JSON  
// @Description    - ecdsa_priv_client (Base64)  // TODO(СДЕЛАЮ ПОПОЗЖЕ(ПОКА ПРОСТО ЗАТЫЧКИ)): во внешнем API передавать зашифрованным паролем  
// @Description    - ks (Base64)                // TODO(СДЕЛАЮ ПОПОЗЖЕ(ПОКА ПРОСТО ЗАТЫЧКИ)): во внешнем API передавать зашифрованным паролем  
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LogInDTO  true  "Email, Password и Platform (web или tg-bot)"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string  "Bad request или неверная платформа"
// @Failure      403   {object}  map[string]string  "incorrect password or email"
// @Failure      404   {object}  map[string]string  "user not found"
// @Failure      500   {string}  string             "Internal Server Error"
// @Router       /users/login [post]
func (h *userHandler) Login(c *gin.Context) {
	const op = "handler.http.users.Login"

	var authDTO dto.LogInDTO

	if err := c.BindJSON(&authDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		log.Printf("Error binding JSON: %v, location %s", err, op)
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

	accessToken, refreshToken, err := h.userService.Login(c, authDTO.Email, authDTO.Password, authDTO.Platform)
	if err != nil {
		if errors.Is(err, serviceUsers.ErrInvalidCredentials) {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusForbidden, gin.H{"error": "incorrect password or email"})
			return
		}
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("Error: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		log.Printf("Error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// TODO: сделать внешнее апи, которое пытается получить эти данные передаются в голом виде
	// (также в другом сервисе будет авторизация через jwt, так что в хедере надо передавать этот access токен)
	ECDSAPrivKeyb64_tg_bot := "ECDSAPrivKeyb64"
	KSb64_tg_bot := "KSb64"

	// TODO: сделать внешнее апи, которое пытается получить эти данные передаются в зашифрованном виде(шифром будет пароль, поэтому нужно еще передавать голый пароль клиента)
	// (также в другом сервисе будет авторизация через jwt, так что в хедере надо передавать этот access токен)
	ECDSAPrivKeyb64_WEB := "ECDSAPrivKeyb64"
	KSb64_WEB := "KSb64"

	// для тг бота
	if authDTO.Platform == "tg-bot" {
		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"refresh_token": refreshToken,
			"ecdsa_priv_client": ECDSAPrivKeyb64_tg_bot,
			"ks": KSb64_tg_bot,
		})
	} else {
		// для web
		utils.SetRefreshTokenCookie(c, refreshToken)

		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"ecdsa_priv_client":ECDSAPrivKeyb64_WEB, 
			"ks": KSb64_WEB,
		})
	}
}
