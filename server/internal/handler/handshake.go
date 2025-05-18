package handler

import (
	"errors"
	"net/http"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/1abobik1/SecureComm/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// @Summary     Инициализация Handshake
// @Description ЗАПРОС ОТ КЛИЕНТА:
// @Description Клиент отправляет свои публичные ключи и nonce1, всё это подписано приватным ECDSA‑ключом.
// @Description Все бинарные данные (ключи, подписи, nonce) закодированы в Base64 (DER для ключей и подписи).
// @Description
// @Description rsa_pub_client - Base64(DER‑закодированный RSA‑публичный ключ клиента)
// @Description ecdsa_pub_client - Base64(DER‑закодированный ECDSA‑публичный ключ клиента)
// @Description nonce1 - Base64(8‑байтовый случайный nonce)
// @Description signature1 - Base64(DER‑закодированная подпись SHA256(clientRSA || clientECDSA || nonce1) приватным ECDSA‑ключом клиента)
// @Description
// @Description ОТВЕТ ОТ СЕРВЕРА:
// @description Сервер отвечает своими публичными ключами и nonce2, всё это подписано приватным ECDSA‑ключом сервера.
// @description Все бинарные данные (ключи, подписи, nonce) закодированы в Base64 (DER для ключей и подписи).
// @description
// @description client_id - SHA256‑хэш от (clientRSA‖clientECDSA), представлен в hex
// @description rsa_pub_server - Base64(DER‑закодированный RSA‑публичный ключ сервера)
// @description ecdsa_pub_server - Base64(DER‑закодированный ECDSA‑публичный ключ сервера)
// @description nonce2 - Base64(8‑байтовый случайный nonce)
// @description signature2 - Base64(DER‑подпись SHA256(rsaServer || ecdsaServer || nonce2 || nonce1 || clientID) приватным ECDSA‑ключом сервера)
// @Tags        handshake
// @Accept      json
// @Produce     json
// @Param       input   body      dto.HandshakeInitReq   true  "Параметры инициации Handshake"
// @Success     200     {object}  dto.HandshakeInitResp  "Успешный ответ сервера"
// @Failure     400     {object}  dto.BadRequestErr      "Некорректный JSON или параметры"
// @Failure     401     {object}  dto.UnauthorizedErr    "Unauthorized или ошибка подписи"
// @Failure     409     {object}  dto.ConflictErr        "Conflict — повторный запрос (replay-detected)"
// @Failure     500     {object}  dto.InternalServerErr  "Внутренняя ошибка сервера"
// @Router      /handshake/init [post]
func (h *handler) Init(c *gin.Context) {
	const op = "location internal.handler.handshake.Init"

	var req dto.HandshakeInitReq

	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}

	rsaPubClient := decodeOrAbort(c, req.RSAPubClient)
	ecdsaPubClient := decodeOrAbort(c, req.ECDSAPubClient)
	nonce1 := decodeOrAbort(c, req.Nonce1)
	sig1 := decodeOrAbort(c, req.Signature1)

	if c.IsAborted() {
		logrus.Errorf("%s: invalid base64 payload", op)
		return
	}

	clientID := h.svc.ComputeFingerprint(c, rsaPubClient, ecdsaPubClient)

	logrus.Infof("created new clientID: %s", clientID)

	serverRSA, serverECDSA, nonce2, sig2, err := h.svc.Init(c, clientID, rsaPubClient, ecdsaPubClient, nonce1, sig1)
	if err != nil {
		if errors.Is(err, service.ErrReplayDetected) {
			logrus.Errorf("Service error: %s", err.Error())
			c.JSON(http.StatusConflict, dto.ConflictErr{Error: err.Error()})
			return
		}
		logrus.Errorf("Service error: %s", err.Error())
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: err.Error()})
		return
	}

	resp := dto.HandshakeInitResp{
		ClientID:       clientID,
		RSAPubServer:   encode(serverRSA),
		ECDSAPubServer: encode(serverECDSA),
		Nonce2:         encode(nonce2),
		Signature2:     encode(sig2),
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary      Завершает Handshake
// @description ЗАПРОС ОТ КЛИЕНТА:
// @description Клиент шлёт RSA-OAEP(encrypted payload), закодированный в Base64.
// @description и signature подписанный payload приватным ключем клиента
// @description об отправляемых полях клиентом encrypted и signature3:
// @description Рандомные 32 байта - это сессионная строка, назовем её ks, которая лежит в payload
// @description payload - это сумма байтов (ks || nonce3 || nonce2)
// @description signature3 - это подписанный payload приватным ключем ECDSA клиента
// @description encrypted - зашифрован RSA-OAEP публичным ключем сервера, отдается в формате Base64
// @Description
// @description ОТВЕТ ОТ СЕРВЕРА:
// @description Сервер возвращает подпись h4 = SHA256(Ks || nonce3 || nonce2), подписанную приватным ECDSA‑ключом сервера и закодированную в Base64.
// @Tags         handshake
// @Accept       json
// @Produce      json
// @Param        X-Client-ID header    string                      true   "Client ID"  default(f44f210d1234abcd...)
// @Param        input       body      dto.HandshakeFinalizeReq    true   "Параметры завершения Handshake"
// @Success      200         {object}  dto.HandshakeFinalizeResp   "Успешный ответ сервера"
// @Failure      400         {object}  dto.BadRequestErr           "Некорректный JSON или параметры"
// @Failure      401         {object}  dto.UnauthorizedErr         "Unauthorized или подпись не верна"
// @Failure      409         {object}  dto.ConflictErr             "Conflict — повторный запрос (replay-detected)"
// @Failure      500         {object}  dto.InternalServerErr       "Внутренняя ошибка сервера"
// @Router       /handshake/finalize [post]
func (h *handler) Finalize(c *gin.Context) {
	var req dto.HandshakeFinalizeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handleBindError(c, err)
		return
	}

	//Base64-декодим encrypted payload
	encrypted, err := decode(req.Encrypted)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	sig3, err := decode(req.Signature3)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// достаем clientID с контекста от middleware
	clientID := c.GetString("clientID")

	sig4, err := h.svc.Finalize(c, clientID, sig3, encrypted)
	if err != nil {
		if errors.Is(err, service.ErrReplayDetected) {
			logrus.Errorf("Service error: %s", err.Error())
			c.JSON(http.StatusConflict, dto.ConflictErr{Error: err.Error()})
			return
		}
		logrus.Errorf("Finalize error for client %s: %v", clientID, err)
		c.JSON(http.StatusUnauthorized, dto.UnauthorizedErr{Error: err.Error()})
		return
	}

	resp := dto.HandshakeFinalizeResp{
		Signature4: encode(sig4),
	}

	c.JSON(http.StatusOK, resp)
}
