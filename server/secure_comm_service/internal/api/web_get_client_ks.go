package api

import (
	"net/http"
	"strconv"

	"github.com/1abobik1/SecureComm/internal/handler/utils"
	"github.com/gin-gonic/gin"
)

type webResp struct {
	KencIV   string `json:"k_enc_iv"`   // base64(ivEnc)
	KencData string `json:"k_enc_data"` // base64(ciphertextEnc)
	KmacIV   string `json:"k_mac_iv"`   // base64(ivMac)
	KmacData string `json:"k_mac_data"` // base64(ciphertextMac)
}

type webReq struct {
	Password string `json:"password"`
}

func (h *WEBClientKeysAPI) GetClientKS(c *gin.Context) {
	var req webReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
		return
	}

	clientID, _ := utils.GetUserID(c)
	kEnc, kMac, err := h.sessionI.GetSessionKeys(c, strconv.Itoa(clientID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no session"})
		return
	}

	// шифруем через AES-GCM, ключ = sha256(req.Password)
	ivEnc, ctEnc, err := encryptAESGCM(req.Password, kEnc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt k_enc failed"})
		return
	}
	ivMac, ctMac, err := encryptAESGCM(req.Password, kMac)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt k_mac failed"})
		return
	}

	c.JSON(http.StatusOK, webResp{
		KencIV:   ivEnc,
		KencData: ctEnc,
		KmacIV:   ivMac,
		KmacData: ctMac,
	})
}
