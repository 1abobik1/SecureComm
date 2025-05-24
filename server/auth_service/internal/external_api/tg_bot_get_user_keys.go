package external_api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

type TgResp struct {
	Kenc string `json:"k_enc"` // base64
	Kmac string `json:"k_mac"` // base64
}

func (c *TGClient) GetClientKS(ctx context.Context, accessToken string) (ks_enc, ks_mac string, er error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL, nil)
	if err != nil {
		logrus.Error(err)
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		logrus.Error(err)
		return "", "", err
	}
	defer resp.Body.Close()

	var tgResp TgResp
	if err := json.NewDecoder(resp.Body).Decode(&tgResp); err != nil {
		logrus.Error(err)
		return "", "", err
	}

	return tgResp.Kenc, tgResp.Kmac, nil
}
