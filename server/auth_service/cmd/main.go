package main

import (
	"net/http"
	"time"

	"github.com/1abobik1/AuthService/config"
	"github.com/1abobik1/AuthService/internal/external_api"
	handlerToken "github.com/1abobik1/AuthService/internal/handler/http/token"
	handlerUsers "github.com/1abobik1/AuthService/internal/handler/http/users"
	"github.com/1abobik1/AuthService/internal/middleware"
	serviceToken "github.com/1abobik1/AuthService/internal/service/token"
	serviceUsers "github.com/1abobik1/AuthService/internal/service/users"
	"github.com/1abobik1/AuthService/internal/storage/postgresql"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	_ "github.com/1abobik1/AuthService/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           File Upload Service API
// @version         1.0
// @description     API для загрузки, получения и удаления файлов …
// @termsOfService  http://example.com/terms/
//
// @contact.name    API Support
// @contact.url     http://www.example.com/support
// @contact.email   support@example.com
//
// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html
//
// @host      localhost:8080
// @BasePath  /
//
// @securityDefinitions.apikey  bearerAuth
// @in                          header
// @name                        Authorization
// @description                 "Bearer {token}"
func main() {
	cfg := config.MustLoad()

	postgresStorage, err := postgresql.NewPostgresStorageProd(cfg.Postgres.StoragePath)
	if err != nil {
		panic("postgres connection error")
	}

	userService := serviceUsers.NewUserService(postgresStorage, *cfg)

	// подключение клиента для внешних апи
	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}
	tgClient := external_api.NewTGClient(cfg.ExternalAPIs.TGClient, httpClient)
	webClient := external_api.NewWEBClient(cfg.ExternalAPIs.WebClient, httpClient)

	userHandler := handlerUsers.NewUserHandler(userService, tgClient, webClient)

	tokenService := serviceToken.NewTokenService(postgresStorage, *cfg)
	tokenHandler := handlerToken.NewTokenHandler(tokenService)

	r := gin.Default()

	// свагер документация(лучше брать доки с папки docs)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// cors conf
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Authorization",
			"Content-Type",
			"X-Orig-Filename",
			"X-Orig-Mime",
			"X-File-Category",
		},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.POST("/user/signup", userHandler.SignUp)
	r.POST("/user/login", middleware.RegistrationAttemptLimiter() ,middleware.NewIPRateLimiter(cfg.LoginLimiter.RPC, cfg.LoginLimiter.Burst, cfg.LoginLimiter.Period), userHandler.Login)
	r.POST("/user/logout", userHandler.Logout)

	r.POST("/token/update", tokenHandler.TokenUpdate)

	if err := r.Run(cfg.HTTPServ.ServerAddr); err != nil {
		panic(err)
	}
}
