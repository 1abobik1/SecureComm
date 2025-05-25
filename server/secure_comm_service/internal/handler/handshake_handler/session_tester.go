package handshake_handler

import (
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/handler/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @Summary     Тестовое расшифрование и проверка целостности сессионного сообщения
// @Description  Клиент шлёт серверу зашифрованный пакет, объединяющий метаданные и payload данные:
// @Description    timestamp - время отправки клиента в формате Unix миллисекунд (8 байт)
// @Description    nonce - криптографически стойкое случайное значение (16 байт) для защиты от replay
// @Description    IV - инициализационный вектор AES-CBC (16 байт)
// @Description    ciphertext - результат AES-256-CBC шифрования сессионного payload'а + PKCS#7
// @Description    tag - HMAC-SHA256 от (IV||ciphertext) для целостности данных
// @Description
// @Description	   по итогу клиент отправляет:
// @Description    encrypted_message = timestamp(8 byte) || nonce(16 byte) || IV(16 byte) || ciphertext || tag(32 byte)
// @Description	   client_signature = Base64(DER‑закодированная подпись SHA256(timestamp || nonce || IV || ciphertext || tag) приватным ECDSA‑ключом клиента
// @Description
// @Description Сервер выполняет следующие шаги:
// @Description    Декодирует Base64 и извлекает blob.
// @Description    Извлекает timestamp и nonce из первых 24 байт и проверяет не использован ли nonce и если diff = (client_timestamp - server_timestamp) > допустимому окну(30секунд). если это так, то запрос отклоняется с ошибкой
// @Description    Получает K_enc и K_mac по clientID, иначе 401 Unauthorized;
// @Description    Проверяет HMAC-SHA256(iv||ciphertext), иначе 401 Unauthorized;
// @Description    Расшифровывает AES-256-CBC, снимает PKCS#7 padding, иначе 400 Bad Request;
// @Description    Возвращает JSON с plaintext - декодированный userData.
// @Tags        session
// @Accept      json
// @Produce     json
// @Param       input            body      dto.SessionMessageReq    true  "Метаданные + зашифрованный payload в Base64"
// @Success     200              {object}  dto.SessionMessageResp   "Успешный ответ: plaintext"
// @Failure     400              {object}  dto.BadRequestErr        "Неверный формат Base64, устаревший timestamp или padding"
// @Failure     401              {object}  dto.UnauthorizedErr      "Session not found или проверка HMAC не прошла"
// @Failure     409              {object}  dto.ConflictErr          "Повторное использование nonce (replay)"
// @Failure     500              {object}  dto.InternalServerErr    "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router      /session/test [post]
func (h *HSHandler) SessionTester(c *gin.Context) {
	var req dto.SessionMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleBindError(c, err)
		return
	}

	clientID, err := utils.GetUserID(c)
	if err != nil {
		logrus.Errorf("GetUserID Errors: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
		return
	}
	clientIDStr := strconv.Itoa(clientID)

	data, err := base64.StdEncoding.DecodeString(req.EncryptedMessage)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.BadRequestErr{Error: "invalid base64 payload"})
		return
	}

	sig, err := utils.Decode(req.ClientSignature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.InternalServerErr{Error: "invalid base64 payload"})
		return
	}

	plaintext, err := h.svc.DecryptWithSession(c, clientIDStr, sig, data)
	if err != nil {
		utils.WriteSessionError(c, err) // обработка различных ошибок
		return
	}

	c.JSON(http.StatusOK, dto.SessionMessageResp{Plaintext: string(plaintext)})
}
