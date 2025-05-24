package api

import (
	"net/http"
	"strconv"

	"github.com/1abobik1/SecureComm/internal/handler/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type tgResp struct {
	Kenc string `json:"k_enc"`
	Kmac string `json:"k_mac"`
}

func (h *TGClientKeysAPI) GetClientKS(c *gin.Context) {
	const op = "internal.api.tg_bot_get_client_ks.GetClientKS"

	clientID, err := utils.GetUserID(c)
	if err != nil {
		logrus.Errorf("GetUserID Errors: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "the user's ID was not found in the token."})
		return
	}
	clientIDStr := strconv.Itoa(clientID)

	kEnc, kMac, err := h.sessionI.GetSessionKeys(c, clientIDStr)
	if err != nil {
		logrus.Errorf("%s: invalid session for clientID %s", op, clientIDStr)
		return
	}

	c.JSON(http.StatusOK, tgResp{
		Kenc: utils.Encode(kEnc),
		Kmac: utils.Encode(kMac),
	})
}
