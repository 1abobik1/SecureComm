package routes

import (
	"github.com/1abobik1/SecureComm/config"
	"github.com/1abobik1/SecureComm/internal/api"
	"github.com/1abobik1/SecureComm/internal/handler/cloud_handler"
	"github.com/1abobik1/SecureComm/internal/handler/handshake_handler"
	"github.com/1abobik1/SecureComm/internal/handler/quota_handler"
	"github.com/1abobik1/SecureComm/internal/middleware"
	tb "github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	toll_gin "github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cfg *config.Config, quotaHandler *quota_handler.QuotaHandler, minioHandler *cloud_handler.MinioHandler, hsHandler *handshake_handler.HSHandler,
	webClient *api.WEBClientKeysAPI, tgClient *api.TGClientKeysAPI,
) {

	authGroup := r.Group("/")
	authGroup.Use(middleware.JWTMiddleware(cfg.JWT.PublicKeyPath))

	{
		// Handshake
		hsLimiter := tb.NewLimiter(cfg.HSLimiter.RPC, &limiter.ExpirableOptions{DefaultExpirationTTL: cfg.HSLimiter.TTL})
		hsLimiter.SetBurst(cfg.HSLimiter.Burst)

		hsGroup := authGroup.Group("/handshake")
		{
			hsGroup.POST("/init", toll_gin.LimitHandler(hsLimiter), hsHandler.Init)
			hsGroup.POST("/finalize", toll_gin.LimitHandler(hsLimiter), hsHandler.Finalize)
		}

		// Session test
		sessionLimiter := tb.NewLimiter(cfg.SesLimiter.RPC, &limiter.ExpirableOptions{DefaultExpirationTTL: cfg.SesLimiter.TTL})
		sessionLimiter.SetBurst(cfg.SesLimiter.Burst)

		sGroup := authGroup.Group("/session")
		{
			sGroup.POST("/test", toll_gin.LimitHandler(sessionLimiter), hsHandler.SessionTester)
		}

		// Файловое API
		routesFileApi := authGroup.Group("/files")
		{
			routesFileApi.POST("/one/encrypted", middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.CreateOneEncrypted)
			routesFileApi.GET("/all", middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.GetAll)
			routesFileApi.GET("/one", middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.GetOne)
			routesFileApi.DELETE("/one", middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.DeleteOne)
			routesFileApi.DELETE("/many", middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.DeleteMany)
		}

		webClientApi := authGroup.Group("/web")
		{
			webClientApi.GET("/ks", webClient.GetClientKS)
		}

		tgClientApi := authGroup.Group("/tg-bot")
		{
			tgClientApi.GET("/ks", tgClient.GetClientKS)
		}

		quotaApi := authGroup.Group("/user")
		{
			quotaApi.POST("/:id/plan/init", quotaHandler.InitUserPlan)
			quotaApi.GET("/:id/usage", quotaHandler.GetUserUsage)
		}
	}
}
