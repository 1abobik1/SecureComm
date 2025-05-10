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
)

func init() {
	binding.EnableDecoderDisallowUnknownFields = true // отвергает лишние поля у DTO при запросе
}

func main() {
	// 1) Загрузка конфига
	cfg := config.MustLoad()

	// Пути к PEM-файлам
	rsaPrivPath := cfg.RSAPrivPath
	rsaPubPath := cfg.RSAPubPath
	ecdsaPrivPath := cfg.ECDSAPrivPath
	ecdsaPubPath := cfg.ECDSAPubPath

	// 2) проверка наличия файлов ключей
	if err := checker.CheckKeys(rsaPrivPath, rsaPubPath, ecdsaPrivPath, ecdsaPubPath); err != nil {
		panic(err)
	}

	// 3) Репозитории
	// 3.1. Серверные ключи из файлов
	serverKeys, err := server_keystore.NewFileKeyStore(
		rsaPrivPath, rsaPubPath,
		ecdsaPrivPath, ecdsaPubPath,
	)
	if err != nil {
		panic(err)
	}

	// 3.2. Redis для клиентских публичных ключей
	clientKeys := client_keystore.NewRedisClientPubKeyStore(
		cfg.RedisServerAddr,
	)

	// 3.3. Redis для хранения nonces (TTL = 5 мин)
	nonceStore := client_noncestore.NewRedisNonceStore(
		cfg.RedisServerAddr,
		cfg.RedisNoncesTTL,
	)

	// 4) Сервисный слой
	hsService := service.NewService(nonceStore, serverKeys, clientKeys)

	// 5) HTTP-Handler
	hsHandler := handler.NewHandler(hsService)

	// 6) Маршрутизация
	r := gin.Default()
	hs := r.Group("/handshake")
	{
		hs.POST("/init", hsHandler.Init)
		hs.POST("/finalize", hsHandler.Finalize)
	}

	// 7) Запуск
	logrus.Infof("Starting server on %s", cfg.HTTPServerAddr)
	if err := r.Run(cfg.HTTPServerAddr); err != nil {
		panic(err)
	}
}
