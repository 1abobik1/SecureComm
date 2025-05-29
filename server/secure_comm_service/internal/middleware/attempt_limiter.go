package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

var (
	// кеш для хранения количества неудачных попыток по IP
	FailedAttemptsCache = cache.New(3*time.Minute, 1*time.Minute)
	mu                  sync.Mutex
	maxFailedAttempts   = 3
	blockDuration       = 3 * time.Minute
)

func RegistrationAttemptLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		// проверка на заблокирован ли IP
		if blockedUntil, found := FailedAttemptsCache.Get("block_" + ip); found {
			// если блокировка еще действует
			if t, ok := blockedUntil.(time.Time); ok && time.Now().Before(t) {
				retryAfter := int(time.Until(t).Seconds())
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":       "too many failed attempts, try again later",
					"retry_after": retryAfter,
				})
				return
			}
		}

		c.Next()

		// после обработки запроса проверяем, был ли неудачный запрос
		// в хендлере на регистрацию при ошибке ставится в контекст "failed_handshake" = true
		if failed, exists := c.Get("failed_handshake"); exists && failed.(bool) {
			mu.Lock()
			defer mu.Unlock()

			key := "fail_" + ip
			countRaw, _ := FailedAttemptsCache.Get(key)
			count := 0
			if countRaw != nil {
				count = countRaw.(int)
			}
			count++
			if count >= maxFailedAttempts {
				// блокируем IP на blockDuration
				FailedAttemptsCache.Set("block_"+ip, time.Now().Add(blockDuration), blockDuration)
				FailedAttemptsCache.Delete(key)
			} else {
				FailedAttemptsCache.Set(key, count, blockDuration)
			}
		}
	}
}
