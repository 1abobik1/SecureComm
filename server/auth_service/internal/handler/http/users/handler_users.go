package handlerUsers

import (
	"context"
)

const ErrValidationEmailOrPassword = "invalid email format or password must be at least 6 characters long"

type UserServiceI interface {
	Register(ctx context.Context, email, password, platform string) (accessJWT string, refreshJWT string, er error)
	Login(ctx context.Context, email, password, platform string) (accessJWT string, refreshJWT string, er error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}

type userHandler struct {
	userService UserServiceI
}

func NewUserHandler(userService UserServiceI) *userHandler {
	return &userHandler{userService: userService}
}
