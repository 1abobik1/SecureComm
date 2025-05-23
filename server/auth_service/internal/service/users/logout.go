package serviceUsers

import "context"

func (s *userService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return s.userStorage.DeleteRefreshToken(ctx, refreshToken)
}
