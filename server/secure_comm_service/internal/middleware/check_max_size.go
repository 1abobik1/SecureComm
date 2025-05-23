package middleware

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

var ErrFileTooLarge = errors.New("file too large")

const MaxFileSize = 5 << 30 // 5 GiB

type countingReader struct {
	R     io.ReadCloser
	read  int64
	limit int64
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.R.Read(p)
	cr.read += int64(n)
	if cr.read > cr.limit {
		return n, ErrFileTooLarge
	}
	return n, err
}

func (cr *countingReader) Close() error {
	return cr.R.Close()
}

func MaxStreamMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = &countingReader{
			R:     c.Request.Body,
			limit: limit,
		}
		c.Next()
	}
}

func MaxSizeMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > limit {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("file is too large: max %d bytes allowed", limit),
			})
			return
		}
		c.Next()
	}
}
