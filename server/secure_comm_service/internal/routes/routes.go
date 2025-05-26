package routes

import (
	"github.com/1abobik1/SecureComm/config"
	"github.com/1abobik1/SecureComm/internal/api"
	"github.com/1abobik1/SecureComm/internal/handler/cloud_handler"
	"github.com/1abobik1/SecureComm/internal/handler/handshake_handler"
	"github.com/1abobik1/SecureComm/internal/handler/quota_handler"
	"github.com/1abobik1/SecureComm/internal/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, cfg *config.Config, quotaHandler *quota_handler.QuotaHandler, minioHandler *cloud_handler.MinioHandler, hsHandler *handshake_handler.HSHandler,
	webClient *api.WEBClientKeysAPI, tgClient *api.TGClientKeysAPI, hsLimiterMiddleware gin.HandlerFunc, sessionLimiterMiddleware gin.HandlerFunc,
) {

	authGroup := r.Group("/")
	authGroup.Use(middleware.JWTMiddleware(cfg.JWT.PublicKeyPath))

	{
		hsGroup := authGroup.Group("/handshake")
		{
			hsGroup.POST("/init", hsLimiterMiddleware, hsHandler.Init)
			hsGroup.POST("/finalize", hsLimiterMiddleware, hsHandler.Finalize)
		}

		sGroup := authGroup.Group("/session")
		{
			sGroup.POST("/test", sessionLimiterMiddleware, hsHandler.SessionTester)
		}

		// Файловое API
		routesFileApi := authGroup.Group("/files")
		{
			routesFileApi.POST("/one/encrypted", sessionLimiterMiddleware, middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.CreateOneEncrypted)
			routesFileApi.GET("/all", sessionLimiterMiddleware, middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.GetAll)
			routesFileApi.GET("/one", sessionLimiterMiddleware, middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.GetOne)
			routesFileApi.DELETE("/one", sessionLimiterMiddleware, middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.DeleteOne)
			routesFileApi.DELETE("/many", sessionLimiterMiddleware, middleware.MaxSizeMiddleware(middleware.MaxFileSize), middleware.MaxStreamMiddleware(middleware.MaxFileSize), minioHandler.DeleteMany)
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
			quotaApi.GET("/:id/usage", sessionLimiterMiddleware, quotaHandler.GetUserUsage)
		}
	}
}
