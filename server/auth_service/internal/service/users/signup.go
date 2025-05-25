package serviceUsers

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/1abobik1/AuthService/internal/external_api"
	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/1abobik1/AuthService/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

func (s *userService) Register(ctx context.Context, email, password, platform string) (accessJWT string, refreshJWT string, er error) {
	const op = "service.users.Register"

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error bcrypt.GenerateFromPassword: %v, location %s \n", err, op)
		return "", "", fmt.Errorf("error bcrypt.GenerateFromPassword: %w", err)
	}

	userID, err := s.userStorage.SaveUser(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Printf("Warning: %v \n", err)
			return "", "", err
		}

		log.Printf("Error failed to save user: %v \n", err)
		return "", "", err
	}

	accessToken, err := utils.CreateAccessToken(userID, s.cfg.AccessTokenTTL, s.cfg.PrivateKeyPath)
	if err != nil {
		log.Printf("Error creating access token: %v \n", err)
		return "", "", fmt.Errorf("error creating access token: %w", err)
	}

	refreshToken, err := utils.CreateRefreshToken(userID, s.cfg.RefreshTokenTTL, s.cfg.PrivateKeyPath)
	if err != nil {
		log.Printf("Error creating refresh token: %v \n", err)
		return "", "", fmt.Errorf("error creating refresh token: %w", err)
	}

	if err := s.userStorage.UpsertRefreshToken(ctx, refreshToken, userID, platform); err != nil {
		log.Printf("Error upserting refresh token in db: %v", err)
		return "", "", fmt.Errorf("error upserting refresh token in db: %w", err)
	}

	if err := external_api.NotifyQuotaService(s.cfg.QuotaServiceURL, userID, accessToken); err != nil {
		log.Printf("warning: failed to init free plan for user %d: %v", userID, err)
		return "", "", fmt.Errorf("failed to init free plan for user %d: %v", userID, err)
	}

	return accessToken, refreshToken, nil
}
