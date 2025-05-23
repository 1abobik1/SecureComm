package handlerUsers

import (
	"context"

	"github.com/1abobik1/AuthService/internal/external_api"
)

const ErrValidation = `the email format is incorrect or the password must be at least 6 characters long. You may have incorrectly specified the "platform" (tg-bot, web).`

type UserServiceI interface {
	Register(ctx context.Context, email, password, platform string) (accessJWT string, refreshJWT string, er error)
	Login(ctx context.Context, email, password, platform string) (accessJWT string, refreshJWT string, er error)
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}

type TGClientKeysI interface {
	GetClientKS(ctx context.Context, accessToken string) (kenc, kmac string, er error)
}

type WEBClientKeysI interface {
	GetClientKS(ctx context.Context, password, accessToken string) (external_api.WebResp, error)
}

type userHandler struct {
	userService UserServiceI
	tgClient    TGClientKeysI
	webClient   WEBClientKeysI
}

func NewUserHandler(userService UserServiceI, tgClient TGClientKeysI, webClient WEBClientKeysI) *userHandler {
	return &userHandler{
		userService: userService,
		tgClient:    tgClient,
		webClient:   webClient,
	}
}
