package serviceToken

import (
	"context"

	"github.com/1abobik1/AuthService/config"
)

type TokenStorageI interface {
	CheckRefreshToken(refreshToken string) (int, error)
	UpdateRefreshToken(oldRefreshToken, newRefreshToken string) error
	GetUserKey(ctx context.Context, userID int) (string, error)
}

type tokenService struct {
	tokenStorage TokenStorageI
	cfg          config.Config
}

func NewTokenService(tokenStorage TokenStorageI, cfg config.Config) *tokenService {
	return &tokenService{
		tokenStorage: tokenStorage,
		cfg:          cfg,
	}
}
