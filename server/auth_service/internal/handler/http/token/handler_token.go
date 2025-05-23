package handlerToken

import "github.com/golang-jwt/jwt/v4"

type TokenSeerviceI interface {
	UpdateAccessToken(refreshToken string) (string, error)
	UpdateRefreshToken(refreshToken string, userID int, userKey string) (string, error)
	ValidateRefreshToken(refreshToken string) (bool, jwt.MapClaims, error)
}

type tokenHandler struct {
	tokenService TokenSeerviceI
}

func NewTokenHandler(tokenService TokenSeerviceI) *tokenHandler {
	return &tokenHandler{tokenService: tokenService}
}
