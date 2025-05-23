package main

import (
	"net/http"
	"time"

	"github.com/1abobik1/AuthService/config"
	"github.com/1abobik1/AuthService/internal/external_api"
	handlerToken "github.com/1abobik1/AuthService/internal/handler/http/token"
	handlerUsers "github.com/1abobik1/AuthService/internal/handler/http/users"
	serviceToken "github.com/1abobik1/AuthService/internal/service/token"
	serviceUsers "github.com/1abobik1/AuthService/internal/service/users"
	"github.com/1abobik1/AuthService/internal/storage/postgresql"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	_ "github.com/1abobik1/AuthService/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.MustLoad()

	postgresStorage, err := postgresql.NewPostgresStorageProd(cfg.StoragePath)
	if err != nil {
		panic("postgres connection error")
	}

	userService := serviceUsers.NewUserService(postgresStorage, *cfg)

	// подключение клиента для внешних апи
	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}
	tgClient := external_api.NewTGClient(cfg.ExternalTGClient, httpClient)
	webClient := external_api.NewWEBClient(cfg.ExternalWebClient, httpClient)

	userHandler := handlerUsers.NewUserHandler(userService, tgClient, webClient)

	tokenService := serviceToken.NewTokenService(postgresStorage, *cfg)
	tokenHandler := handlerToken.NewTokenHandler(tokenService)

	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // свагер документация

	// cors conf
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/user/signup", userHandler.SignUp)
	r.POST("/user/login", userHandler.Login)
	r.POST("/user/logout", userHandler.Logout)

	r.POST("/token/update", tokenHandler.TokenUpdate)

	if err := r.Run(cfg.HTTPServer); err != nil {
		panic(err)
	}
}
