package handler

import (
	"net/http"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Init обработчик /handshake/init
func (h *handler) Init(c *gin.Context) {
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
		return
	}

	clientID := h.svc.ComputeFingerprint(c, rsaPubClient, ecdsaPubClient)

	serverRSA, serverECDSA, nonce2, sig2, err := h.svc.Init(c, clientID, rsaPubClient, ecdsaPubClient, nonce1, sig1)
	if err != nil {
		logrus.Errorf("Service error: %s", err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()}) // не забыть бы убрать err.Error()
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

// Init обработчик /handshake/finalize
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

	// достаем clientID с контекста от middleware
	clientID := c.GetString("clientID")

	sig4, err := h.svc.Finalize(c, clientID, encrypted)
	if err != nil {
		logrus.Errorf("Finalize error for client %s: %v", clientID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	resp := dto.HandshakeFinalizeResp{
		Signature4: encode(sig4),
	}

	c.JSON(http.StatusOK, resp)
}
