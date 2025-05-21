package serviceUsers

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/1abobik1/AuthService/internal/storage"
	"github.com/1abobik1/AuthService/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

func (s *userService) Login(ctx context.Context, email, password, platform string) (accessJWT string, refreshJWT string, er error) {
	const op = "service.users.Login"

	userModel, err := s.userStorage.FindUser(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("Warning: %v, location %s", err, op)
			return "", "", err
		}
		log.Printf("Error failed to save user: %v, location %s", err, op)
		return "", "", err
	}

	if err := bcrypt.CompareHashAndPassword(userModel.Password, []byte(password)); err != nil {
		log.Printf("Wrong password: %v, location %s", err, op)
		return "", "", ErrInvalidCredentials
	}

	accessToken, err := utils.CreateAccessToken(userModel.ID, s.cfg.AccessTokenTTL, s.cfg.PrivateKeyPath)
	if err != nil {
		log.Printf("Error creating access token: %v, location %s \n", err, op)
		return  "", "", fmt.Errorf("error creating access token: %w", err)
	}

	refreshToken, err := utils.CreateRefreshToken(userModel.ID, s.cfg.RefreshTokenTTL, s.cfg.PrivateKeyPath)
	if err != nil {
		log.Printf("Error creating refresh token: %v, location %s \n", err, op)
		return "", "", fmt.Errorf("error creating refresh token: %w", err)
	}

	if err := s.userStorage.UpsertRefreshToken(ctx, refreshToken, userModel.ID, platform); err != nil {
		log.Printf("Error upserting refresh token in db: %v", err)
		return "", "", fmt.Errorf("error upserting refresh token in db: %w", err)
	}

	return accessToken, refreshToken, nil
}
