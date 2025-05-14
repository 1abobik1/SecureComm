package main

import (
	"github.com/1abobik1/SecureComm/config"
	"github.com/1abobik1/SecureComm/internal/checker"
	"github.com/1abobik1/SecureComm/internal/handler"
	"github.com/1abobik1/SecureComm/internal/middleware"
	"github.com/1abobik1/SecureComm/internal/repository/client_keystore"
	"github.com/1abobik1/SecureComm/internal/repository/client_noncestore"
	"github.com/1abobik1/SecureComm/internal/repository/server_keystore"
	"github.com/1abobik1/SecureComm/internal/repository/session_store"

	"github.com/1abobik1/SecureComm/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"

	tb "github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	toll_gin "github.com/didip/tollbooth_gin"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/1abobik1/SecureComm/docs"
)

func init() {
	binding.EnableDecoderDisallowUnknownFields = true // отвергает лишние поля у запроса
}

// @title           SecureComm API
// @version         1.0
// @description     Документация о внутренней реализации и логики работы находится в папке docs
// @host      localhost:8080
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

	// redis для клиентских публичных ключей
	clientKeys := client_keystore.NewRedisClientPubKeyStore(
		cfg.Redis.ServerAddr,
	)

	// redis для хранения nonces
	nonceStore := client_noncestore.NewRedisNonceStore(
		cfg.Redis.ServerAddr,
		cfg.Redis.NoncesTTL,
	)

	// redis для хранения сессионных строк
	sessionStore := session_store.NewRedisSessionStore(
		cfg.Redis.ServerAddr,
		cfg.Redis.SessionKeyTTL,
	)

	// сервисный слой
	hsService := service.NewService(nonceStore, serverKeys, clientKeys, sessionStore)

	// хендлерный слой
	hsHandler := handler.NewHandler(hsService)

	// limiter для /handshake
	hsLimiter := tb.NewLimiter(cfg.HSLimiter.RPC, &limiter.ExpirableOptions{DefaultExpirationTTL: cfg.HSLimiter.TTL})
	hsLimiter.SetBurst(cfg.HSLimiter.Burst)
	// limiter сначала пробует сделать лимит по client_id, если его нет в header, то по ip
	hsLimiter.SetIPLookups([]string{
		"Header:X-Client-ID",
		"RemoteAddr",
	})

	// маршрутизация
	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // свагер документация
	
	hs := r.Group("/handshake")
	{
		hs.POST("/init", toll_gin.LimitHandler(hsLimiter), hsHandler.Init)
		hs.POST("/finalize", middleware.RequireClientID(), toll_gin.LimitHandler(hsLimiter), hsHandler.Finalize)
	}

	// запуск серва
	logrus.Infof("Starting server on %s", cfg.HTTPServ.ServerAddr)
	if err := r.Run(cfg.HTTPServ.ServerAddr); err != nil {
		panic(err)
	}
}
