package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// Decode из base64 в байты
func decode(s string) ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		logrus.Errorf("base64 decode error: %s", err)
		return []byte{}, err
	}
	return b, nil
}

// DecodeOrAbort пробует декодировать base64-строку s функцией decode,
// и при ошибке отдаёт клиенту 500 и прерывает обработчик.
func decodeOrAbort(c *gin.Context, s string) []byte {
	b, err := decode(s)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.InternalServerErr{Error: "invalid base64 payload"})
		c.Abort()
		return nil
	}
	return b
}

// Encode кодирует байты в base64
func encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func handleBindError(c *gin.Context, err error) {

	if verrs, ok := err.(validator.ValidationErrors); ok {
		out := make(map[string]string, len(verrs))

		for _, fe := range verrs {
			out[fe.Field()] = fmt.Sprintf("must satisfy %s", fe.Tag())
		}

		logrus.WithError(err).Warn(out)
		c.JSON(http.StatusBadRequest, gin.H{"errors": out})
		return
	}

	logrus.WithError(err).Warn("invalid request data")
	c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "invalid request data"})
}

func writeSessionError(c *gin.Context, err error) {
	switch err {
	case nil:
		return
	case service.ErrInvalidSession:
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: "session not found"})
	case service.ErrInvalidPayload:
		c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "invalid encrypted payload"})
	case service.ErrBadMAC:
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: "message authentication failed"})
	case service.ErrStaleTimestamp:
		c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "stale timestamp"})
	case service.ErrReplayDetected:
		c.JSON(http.StatusConflict, dto.ConflictErr{Error: "replay detected"})
	default:
		c.JSON(http.StatusInternalServerError, dto.InternalServerErr{Error: err.Error()})
	}
}
