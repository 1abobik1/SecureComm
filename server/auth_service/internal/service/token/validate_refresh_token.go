package serviceToken

import (
	"errors"
	"log"

	"github.com/1abobik1/AuthService/pkg/auth/validation"
	"github.com/golang-jwt/jwt/v4"
)

// use only for refresh_token
func (s *tokenService) ValidateRefreshToken(refreshToken string) (bool, jwt.MapClaims, error) {
	const op = "service.token.validation.ValidateRefreshToken"

	claims, err := validation.ValidateToken(refreshToken, s.cfg.JWT.PublicKeyPath)
	if err != nil {
		if errors.Is(err, validation.ErrTokenExpired) {
			log.Printf("Warning: token expired, location: %s", op)
			return true, claims, nil // Токен истёк, но валиден
		}
		log.Printf("Error: %v, location: %s", err, op)
		return false, nil, err // Токен невалиден
	}

	return false, claims, nil // Токен валиден и не истёк
}
