package utils

import (
	"context"
	"fmt"

	"encoding/base64"
	"net/http"

	"github.com/golang-jwt/jwt/v4"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/service/handshake_service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// GetUserID извлекает user_id из контекста
func GetUserID(ctx context.Context) (int, error) {
	claims, ok := ctx.Value("claims").(jwt.MapClaims)
	if !ok {
		return -1, fmt.Errorf("не удалось извлечь claims")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return -1, fmt.Errorf("не найден user_id в токене")
	}

	return int(userIDFloat), nil
}

// Decode из base64 в байты
func Decode(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		logrus.Errorf("base64 decode error: %s", err)
		return []byte{}, err
	}
	return b, nil
}

// DecodeOrAbort пробует декодировать base64-строку s функцией decode,
// и при ошибке отдаёт клиенту 500 и прерывает обработчик.
func DecodeOrAbort(c *gin.Context, s string) []byte {
	b, err := Decode(s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.InternalServerErr{Error: "invalid base64 payload"})
		c.Abort()
		return nil
	}
	return b
}

// Encode кодирует байты в base64
func Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func HandleBindError(c *gin.Context, err error) {

	if verrs, ok := err.(validator.ValidationErrors); ok {
		out := make(map[string]string, len(verrs))

		for _, fe := range verrs {
			out[fe.Field()] = fmt.Sprintf("must satisfy %s", fe.Tag())
		}

		logrus.WithError(err).Warn(out)
		c.Set("failed_registration", true)
		c.JSON(http.StatusBadRequest, gin.H{"errors": out})
		return
	}

	logrus.WithError(err).Warn("invalid request data")
	c.Set("failed_registration", true)
	c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "invalid request data"})
}

func WriteSessionError(c *gin.Context, err error) {
	switch err {
	case nil:
		return
	case handshake_service.ErrInvalidSession:
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: "session not found"})
	case handshake_service.ErrInvalidPayload:
		c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "invalid encrypted payload"})
	case handshake_service.ErrBadMAC:
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: "message authentication failed"})
	case handshake_service.ErrStaleTimestamp:
		c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "stale timestamp"})
	case handshake_service.ErrReplayDetected:
		c.JSON(http.StatusConflict, dto.ConflictErr{Error: "replay detected"})
	default:
		c.JSON(http.StatusInternalServerError, dto.InternalServerErr{Error: err.Error()})
	}
}
