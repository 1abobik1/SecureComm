package external_api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

type webReq struct {
	Password string `json:"password"`
}

type WebResp struct {
	KencIV   string `json:"k_enc_iv"`   // base64(ivEnc)
	KencData string `json:"k_enc_data"` // base64(ciphertextEnc)
	KmacIV   string `json:"k_mac_iv"`   // base64(ivMac)
	KmacData string `json:"k_mac_data"` // base64(ciphertextMac)
}

func (c *WEBClient) GetClientKS(ctx context.Context, password, accessToken string) (WebResp, error) {
	payload := webReq{
		Password: password,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return WebResp{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		logrus.Error(err)
		return WebResp{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return WebResp{}, err
	}
	defer resp.Body.Close()

	var webResp WebResp
	if err := json.NewDecoder(resp.Body).Decode(&webResp); err != nil {
		logrus.Error(err)
		return WebResp{}, err
	}

	return webResp, nil
}
