package main

import (
	"context"
	"log"
	"time"

	"github.com/1abobik1/SecureComm/config"
	"github.com/1abobik1/SecureComm/internal/api"
	"github.com/1abobik1/SecureComm/internal/checker"
	"github.com/1abobik1/SecureComm/internal/handler/cloud_handler"
	"github.com/1abobik1/SecureComm/internal/handler/handshake_handler"
	"github.com/1abobik1/SecureComm/internal/handler/quota_handler"
	"github.com/1abobik1/SecureComm/internal/middleware"
	"github.com/1abobik1/SecureComm/internal/repository/client_keystore"
	"github.com/1abobik1/SecureComm/internal/repository/nonce_store"
	"github.com/1abobik1/SecureComm/internal/repository/server_keystore"
	"github.com/1abobik1/SecureComm/internal/repository/session_store"
	"github.com/1abobik1/SecureComm/internal/routes"
	"github.com/1abobik1/SecureComm/internal/service/cloud_service"
	"github.com/1abobik1/SecureComm/internal/service/handshake_service"
	"github.com/1abobik1/SecureComm/internal/service/quota_service"
	"github.com/gin-contrib/cors"
	"github.com/go-redis/redis/v8"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"

	_ "github.com/1abobik1/SecureComm/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func init() {
	binding.EnableDecoderDisallowUnknownFields = true // отвергает лишние поля у запроса
}

// @title           SecureComm API
// @version         1.0
// @description     Документация о внутренней реализации и логики работы находится в папке docs
// @host      localhost:8080
//
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                "Bearer {token}"
func main() {
	// загрузка конфига
	cfg := config.MustLoad()

	// пути к ключам сервера
	rsaPrivPath := cfg.ServKeys.RSAPrivPath
	rsaPubPath := cfg.ServKeys.RSAPubPath
	ecdsaPrivPath := cfg.ServKeys.ECDSAPrivPath
	ecdsaPubPath := cfg.ServKeys.ECDSAPubPath

	// проверка наличия файлов ключей
	if err := checker.CheckKeys(rsaPrivPath, rsaPubPath, ecdsaPrivPath, ecdsaPubPath); err != nil {
		panic(err)
	}

	// серверные ключи из файлов
	serverKeys, err := server_keystore.NewFileKeyStore(
		rsaPrivPath, rsaPubPath,
		ecdsaPrivPath, ecdsaPubPath,
	)
	if err != nil {
		panic(err)
	}

	// redis
	rClient := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.ServerAddr,
	})
	if err := rClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis connection error: %v", err)
	}

	// redis для клиентских публичных ключей
	clientKeys := client_keystore.NewRedisClientPubKeyStore(
		rClient,
		cfg.Redis.ClientPubKeysTTL,
	)

	// redis для хранения nonces для handshake
	hsNonceStore := nonce_store.NewRedisNonceStore(
		rClient,
		cfg.Redis.HandshakeNoncesTTL,
	)

	// redis для хранения nonces после handshake при обмене сообщениями
	sesNonceStore := nonce_store.NewRedisSessionNonceStore(
		rClient,
		cfg.Redis.SessionNoncesTTL,
	)

	// redis для хранения сессионных строк
	sessionStore := session_store.NewRedisSessionStore(
		rClient,
		cfg.Redis.SessionKeyTTL,
	)

	// Инициализация MinIO cloud_service слой
	minioService := cloud_service.NewMinioClient(*cfg, rClient)
	if err := minioService.InitMinio(cfg.Minio.Port, cfg.Minio.RootUser, cfg.Minio.RootPassword, cfg.Minio.UseSSL); err != nil {
		log.Fatalf("minio init error: %v", err)
	}

	// сервисный слой qouta
	quotaService, err := quota_service.NewQuotaService(cfg.Postges.StoragePath)
	if err != nil {
		panic(err)
	}

	// хендлерный слой quota
	quotaHandler := quota_handler.NewQuotaHandler(quotaService)
	// хендлерный слой cloud_handler
	minioHandler := cloud_handler.NewMinioHandler(minioService, quotaService)
	// сервисный слой handshake
	hsService := handshake_service.NewService(hsNonceStore, sesNonceStore, serverKeys, clientKeys, sessionStore)
	// хендлерный слой handshake
	hsHandler := handshake_handler.NewHandler(hsService)
	// внешние клиенты
	webClient := api.NewWEBClientKeysAPI(sessionStore)
	tgClient := api.NewTGClientKeysAPI(sessionStore)

	// маршрутизация
	r := gin.Default()

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

	// свагер документация(лучше брать доки с папки docs)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	hsLimiter := middleware.NewIPRateLimiter(cfg.HSLimiter.RPC, cfg.HSLimiter.Burst, cfg.HSLimiter.Period)         // middleware limiter для /handshake
	hsAttemptLimiter := middleware.RegistrationAttemptLimiter()
	sessionLimiter := middleware.NewIPRateLimiter(cfg.SesLimiter.RPC, cfg.SesLimiter.Burst, cfg.SesLimiter.Period) // middleware limiter для остальных апи
	// регистрация всех маршрутов
	routes.RegisterRoutes(r, cfg, quotaHandler, minioHandler, hsHandler, webClient, tgClient, hsLimiter, sessionLimiter, hsAttemptLimiter)

	logrus.Infof("Starting server on %s", cfg.HTTPServ.ServerAddr)
	if err := r.Run(cfg.HTTPServ.ServerAddr); err != nil {
		panic(err)
	}

	// запуск серва
	logrus.Infof("Starting server on %s", cfg.HTTPServ.ServerAddr)
	if err := r.Run(cfg.HTTPServ.ServerAddr); err != nil {
		panic(err)
	}
}
