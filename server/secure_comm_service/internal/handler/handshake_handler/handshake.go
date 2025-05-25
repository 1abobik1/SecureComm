package handshake_handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/handler/utils"
	"github.com/1abobik1/SecureComm/internal/service/handshake_service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @Summary     Инициализация Handshake
// @Description ЗАПРОС ОТ КЛИЕНТА:
// @Description Клиент отправляет свои публичные ключи и nonce1, всё это подписано приватным ECDSA-ключом.
// @Description Все бинарные данные (ключи, подписи, nonce) закодированы в Base64 (DER для ключей и подписи).
// @Description
// @Description rsa_pub_client - Base64(DER-закодированный RSA-публичный ключ клиента)
// @Description ecdsa_pub_client - Base64(DER-закодированный ECDSA-публичный ключ клиента)
// @Description nonce1 - Base64(8-байтовый случайный nonce)
// @Description signature1 - Base64(DER-закодированная подпись SHA256(clientRSA || clientECDSA || nonce1) приватным ECDSA-ключом клиента)
// @Description
// @Description ОТВЕТ ОТ СЕРВЕРА:
// @Description Сервер отвечает своими публичными ключами и nonce2, всё это подписано приватным ECDSA-ключом сервера.
// @Description Все бинарные данные (ключи, подписи, nonce) закодированы в Base64 (DER для ключей и подписи).
// @Description
// @Description client_id будет извлечён из JWT (Bearer-токена).
// @Description rsa_pub_server - Base64(DER-закодированный RSA-публичный ключ сервера)
// @Description ecdsa_pub_server - Base64(DER-закодированный ECDSA-публичный ключ сервера)
// @Description nonce2 - Base64(8-байтовый случайный nonce)
// @Description signature2 - Base64(DER-подпись SHA256(rsaServer || ecdsaServer || nonce2 || nonce1 || clientID) приватным ECDSA-ключом сервера)
// @Tags        handshake
// @Accept      json
// @Produce     json
// @Param       input   body      dto.HandshakeInitReq   true  "Параметры инициации Handshake"
// @Success     200     {object}  dto.HandshakeInitResp  "Успешный ответ сервера"
// @Failure     400     {object}  dto.BadRequestErr      "Некорректный JSON или параметры"
// @Failure     401     {object}  dto.UnauthorizedErr    "Unauthorized или ошибка подписи"
// @Failure     409     {object}  dto.ConflictErr        "Conflict — повторный запрос (replay-detected)"
// @Failure     500     {object}  dto.InternalServerErr  "Внутренняя ошибка сервера"
// @Security    bearerAuth
// @Router      /handshake/init [post]
func (h *HSHandler) Init(c *gin.Context) {
	const op = "location internal.handler.handshake.Init"

	var req dto.HandshakeInitReq

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleBindError(c, err)
		return
	}

	rsaPubClient := utils.DecodeOrAbort(c, req.RSAPubClient)
	ecdsaPubClient := utils.DecodeOrAbort(c, req.ECDSAPubClient)
	nonce1 := utils.DecodeOrAbort(c, req.Nonce1)
	sig1 := utils.DecodeOrAbort(c, req.Signature1)

	if c.IsAborted() {
		logrus.Errorf("%s: invalid base64 payload", op)
		return
	}

	clientID, err := utils.GetUserID(c)
	if err != nil {
		logrus.Errorf("GetUserID Errors: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
		return
	}
	clientIDStr := strconv.Itoa(clientID)

	logrus.Infof("created new clientID: %s", clientIDStr)

	serverRSA, serverECDSA, nonce2, sig2, err := h.svc.Init(c, clientIDStr, rsaPubClient, ecdsaPubClient, nonce1, sig1)
	if err != nil {
		if errors.Is(err, handshake_service.ErrReplayDetected) {
			logrus.Errorf("Service error: %s", err.Error())
			c.JSON(http.StatusConflict, dto.ConflictErr{Error: err.Error()})
			return
		}
		logrus.Errorf("Service error: %s", err.Error())
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: err.Error()})
		return
	}

	resp := dto.HandshakeInitResp{
		ClientID:       clientIDStr,
		RSAPubServer:   utils.Encode(serverRSA),
		ECDSAPubServer: utils.Encode(serverECDSA),
		Nonce2:         utils.Encode(nonce2),
		Signature2:     utils.Encode(sig2),
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary      Завершает Handshake
// @Description  ЗАПРОС ОТ КЛИЕНТА:
// @Description  Клиент шлёт RSA-OAEP(encrypted payload), закодированный в Base64.
// @Description  и signature подписанный payload приватным ключем клиента
// @Description  об отправляемых полях клиентом encrypted и signature3:
// @Description  Рандомные 32 байта - это сессионная строка, назовем её ks, которая лежит в payload
// @Description  payload - это сумма байтов (ks || nonce3 || nonce2)
// @Description  signature3 - это подписанный payload приватным ключем ECDSA клиента в base64
// @Description
// @Description  ОТВЕТ ОТ СЕРВЕРА:
// @Description  Сервер возвращает подпись signature4 = SHA256(Ks || nonce3 || nonce2), подписанную приватным ECDSA-ключом сервера и закодированную в Base64.
// @Tags         handshake
// @Accept       json
// @Produce      json
// @Param        input       body      dto.HandshakeFinalizeReq    true   "Параметры завершения Handshake"
// @Success      200         {object}  dto.HandshakeFinalizeResp   "Успешный ответ сервера"
// @Failure      400         {object}  dto.BadRequestErr           "Некорректный JSON или параметры"
// @Failure      401         {object}  dto.UnauthorizedErr         "Unauthorized или подпись не верна"
// @Failure      409         {object}  dto.ConflictErr             "Conflict — повторный запрос (replay-detected)"
// @Failure      500         {object}  dto.InternalServerErr       "Внутренняя ошибка сервера"
// @Security     bearerAuth
// @Router       /handshake/finalize [post]
func (h *HSHandler) Finalize(c *gin.Context) {
	var req dto.HandshakeFinalizeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.HandleBindError(c, err)
		return
	}

	//Base64-декодим encrypted payload
	encrypted, err := utils.Decode(req.Encrypted)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	sig3, err := utils.Decode(req.Signature3)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	clientID, err := utils.GetUserID(c)
	if err != nil {
		logrus.Errorf("GetUserID Errors: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
		return
	}
	clientIDStr := strconv.Itoa(clientID)

	sig4, err := h.svc.Finalize(c, clientIDStr, sig3, encrypted)
	if err != nil {
		if errors.Is(err, handshake_service.ErrReplayDetected) {
			logrus.Errorf("Service error: %s", err.Error())
			c.JSON(http.StatusConflict, dto.ConflictErr{Error: err.Error()})
			return
		}
		logrus.Errorf("finalize error for client %s: %v", clientIDStr, err)
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: err.Error()})
		return
	}

	resp := dto.HandshakeFinalizeResp{
		Signature4: utils.Encode(sig4),
	}

	c.JSON(http.StatusOK, resp)
}
