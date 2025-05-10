package handler

import (
	"net/http"

	"github.com/1abobik1/SecureComm/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// интерфейс бизнес-логики handshake
type Service interface {
	Init(clientID string, clientRSA, clientECDSA []byte, nonce1 []byte, sig1 []byte) (serverRSA, serverECDSA, nonce2, signature2 []byte, err error)
	ComputeFingerprint(rsaPub, ecdsaPub []byte) string
	Finalize(encrypted []byte) (signature4 []byte, err error)
}

type handler struct {
	svc Service
}

func NewHandler(svc Service) *handler {
	return &handler{svc: svc}
}

// Init обработчик /handshake/init
func (h *handler) Init(c *gin.Context) {
	var req dto.HandshakeReq

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

	clientID := h.svc.ComputeFingerprint(rsaPubClient, ecdsaPubClient)

	serverRSA, serverECDSA, nonce2, sig2, err := h.svc.Init(clientID, rsaPubClient, ecdsaPubClient, nonce1, sig1)
	if err != nil {
		logrus.Errorf("Service error: %s", err.Error())
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()}) // не забыть бы убрать err.Error()
		return
	}

	resp := dto.HandshakeResp{
		ClientID:       clientID,
		RSAPubServer:   encode(serverRSA),
		ECDSAPubServer: encode(serverECDSA),
		Nonce2:         encode(nonce2),
		Signature2:     encode(sig2),
	}

	c.JSON(http.StatusOK, resp)
}

func (h *handler) Finalize(c *gin.Context) {
	panic("inplement meeee")
}
