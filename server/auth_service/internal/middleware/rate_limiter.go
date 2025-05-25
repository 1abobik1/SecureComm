package middleware

import (
	"time"

	"github.com/didip/tollbooth/v7"
	toll_limiter "github.com/didip/tollbooth/v7/limiter"
	"github.com/gin-gonic/gin"
)

// NewIPRateLimiter возвращает gin.HandlerFunc, который ограничивает
// requests по IP: maxReqs запросов за period, burst — мгновенный всплеск.
func NewIPRateLimiter(maxReqs float64, burst int, period time.Duration) gin.HandlerFunc {
	lim := tollbooth.NewLimiter(maxReqs, &toll_limiter.ExpirableOptions{
		DefaultExpirationTTL: period,
	})
	lim.SetBurst(burst)
	lim.SetIPLookups([]string{"RemoteAddr"})

	return func(c *gin.Context) {
		if httpErr := tollbooth.LimitByRequest(lim, c.Writer, c.Request); httpErr != nil {
			retryAfter := c.Writer.Header().Get("Retry-After")
			c.AbortWithStatusJSON(httpErr.StatusCode, gin.H{
				"error":       "too many attempts, try again later",
				"retry_after": retryAfter,
			})
			return
		}
		c.Next()
	}
}
