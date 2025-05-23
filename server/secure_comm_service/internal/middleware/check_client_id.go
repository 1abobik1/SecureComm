package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

// проверяет наличие заголовка X-Client-ID в запросе.
// сохраняет clientID в контексте
func RequireClientID() gin.HandlerFunc {
    return func(c *gin.Context) {
        clientID := c.GetHeader("X-Client-ID")
        if len(clientID) == 0 {
            logrus.Errorf("missing X-Client-ID header")
            c.JSON(http.StatusBadRequest, gin.H{"error": "missing X-Client-ID header"})
            c.Abort()
            return
        }
        c.Set("clientID", clientID)
        c.Next()
    }
}
