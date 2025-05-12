package main

import (
	"github.com/1abobik1/SecureComm/config"
	"github.com/1abobik1/SecureComm/internal/checker"
	"github.com/1abobik1/SecureComm/internal/handler"
	"github.com/1abobik1/SecureComm/internal/repository/client_keystore"
	"github.com/1abobik1/SecureComm/internal/repository/client_noncestore"
	"github.com/1abobik1/SecureComm/internal/repository/server_keystore"

	"github.com/1abobik1/SecureComm/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"

	tb "github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	toll_gin "github.com/didip/tollbooth_gin"
)

func init() {
	binding.EnableDecoderDisallowUnknownFields = true // отвергает лишние поля у запроса
}

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

	// Redis для клиентских публичных ключей
	clientKeys := client_keystore.NewRedisClientPubKeyStore(
		cfg.Redis.RedisServerAddr,
	)

	// Redis для хранения nonces (TTL = 10 мин)
	nonceStore := client_noncestore.NewRedisNonceStore(
		cfg.Redis.RedisServerAddr,
		cfg.Redis.RedisNoncesTTL,
	)

	// сервисный слой
	hsService := service.NewService(nonceStore, serverKeys, clientKeys)

	// HTTP-Handler
	hsHandler := handler.NewHandler(hsService)

	// limiter для /handshake
	hsLimiter := tb.NewLimiter(cfg.HSLimiter.RPC, &limiter.ExpirableOptions{DefaultExpirationTTL: cfg.HSLimiter.TTL})
	hsLimiter.SetBurst(cfg.HSLimiter.Burst)

	// маршрутизация
	r := gin.Default()
	hs := r.Group("/handshake")
	{
		hs.POST("/init", toll_gin.LimitHandler(hsLimiter), hsHandler.Init)
		hs.POST("/finalize", toll_gin.LimitHandler(hsLimiter), hsHandler.Finalize)
	}

	// запуск серва
	logrus.Infof("Starting server on %s", cfg.HTTPServ.ServerAddr)
	if err := r.Run(cfg.HTTPServ.ServerAddr); err != nil {
		panic(err)
	}
}
