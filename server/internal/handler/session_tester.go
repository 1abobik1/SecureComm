package handler

import (
	"encoding/base64"
	"net/http"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @Summary     Тестовое расшифрование сессионного сообщения
// @Description
// @Description  Клиент отправляет зашифрованное Base64‑сообщение,
// @Description  сервер пытается расшифровать его текущим сессионным ключом, сохранённым по Client-ID.
// @Description  Если расшифровка прошла успешно, возвращает plaintext, иначе — ошибку.
// @Description
// @Tags        session
// @Accept      json
// @Produce     json
// @Param       X-Client-ID      header    string               true  "Client ID"
// @Param       input            body      dto.SessionTestReq   true  "Тестовое зашифрованное сообщение"
// @Success     200              {object}  dto.SessionTestResp  "Расшифрованный текст"
// @Failure     400              {object}  dto.BadRequestErr    "Некорректный Base64 или параметры"
// @Failure     401              {object}  dto.UnauthorizedErr  "Не удалось расшифровать сообщение (invalid session/key)"
// @Failure     500              {object}  dto.InternalServerErr "Внутренняя ошибка сервера"
// @Router      /session/test [post]
func (h *handler) SessionTester(c *gin.Context) {
	var req dto.SessionTestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}

	clientID := c.GetString("clientID")
	data, err := base64.StdEncoding.DecodeString(req.EncryptedMessage)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "invalid base64 payload"})
		return
	}

	plaintext, err := h.svc.DecryptWithSession(c, clientID, data)
	switch err {
	case nil:
		logrus.Infof("Server decrypted: %s", plaintext)
		c.JSON(http.StatusOK, dto.SessionTestResp{Plaintext: string(plaintext)})
	case service.ErrInvalidSession:
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: "session not found"})
	case service.ErrInvalidPayload:
		c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "invalid encrypted payload"})
	case service.ErrBadMAC:
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: "message authentication failed"})
	default:
		c.JSON(http.StatusInternalServerError, dto.InternalServerErr{Error: err.Error()})
	}
}
