package external_api

import (
	"fmt"
	"net/http"
	"time"
)

// notifyQuotaService делает POST /users/{id}/plan/init и вставляет Bearer-токен
func NotifyQuotaService(baseURL string, userID int, accessToken string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s/user/%d/plan/init", baseURL, userID)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	//  заголовок авторизации
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request to quota service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("quota service returned status %d", resp.StatusCode)
	}
	return nil
}
