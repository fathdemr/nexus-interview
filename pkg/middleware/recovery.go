package middleware

import (
	"net/http"

	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/fathdemr/nexus-interview/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery returns a Gin middleware that catches panics, logs the stack trace,
// and responds with a 500 rather than crashing the process.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.L().Error("panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					httputil.NewErrorResponse("internal server error", "INTERNAL_ERROR"))
			}
		}()
		c.Next()
	}
}
