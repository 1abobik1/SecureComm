package handlerUsers

import (
	"context"
)

const ErrValidation = `the email format is incorrect or the password must be at least 6 characters long. You may have incorrectly specified the "platform" (tg-bot, web).`

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
