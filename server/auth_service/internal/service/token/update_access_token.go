package serviceToken

import (
	"fmt"
	"log"

	"github.com/1abobik1/AuthService/internal/utils"
)

func (s *tokenService) UpdateAccessToken(refreshToken string) (string, error) {
	const op = "service.token.refresh.UpdateAccessToken"

	userID, err := s.tokenStorage.CheckRefreshToken(refreshToken)
	if err != nil {
		log.Printf("Error: %v", err)
		return "", err
	}

	newAccessToken, err := utils.CreateAccessToken(userID, s.cfg.JWT.AccessTokenTTL, s.cfg.JWT.PrivateKeyPath)
	if err != nil {
		log.Printf("Error creating access token: %v, location %s \n", err, op)
		return "", fmt.Errorf("error creating access token: %w", err)
	}

	return newAccessToken, err
}
