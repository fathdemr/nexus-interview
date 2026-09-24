package middleware

import (
	"time"

	"github.com/fathdemr/nexus-interview/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger returns a Gin middleware that logs each request with method,
// path, status, latency, client IP, and optionally the request/response bodies
// when a handler sets "x-log-enable" = true on the context.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.String("raw_path", c.Request.URL.RequestURI()),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		}

		// Attach request/response bodies only when the handler explicitly enables logging
		if enabled, _ := c.Get("x-log-enable"); enabled == true {
			if reqBody, exists := c.Get("x-log-request-body"); exists && reqBody != nil {
				fields = append(fields, zap.Any("request_body", reqBody))
			}
			if resBody, exists := c.Get("x-log-response-body"); exists && resBody != nil {
				fields = append(fields, zap.Any("response_body", resBody))
			}
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case status >= 500:
			logger.L().Error("request", fields...)
		case status >= 400:
			logger.L().Warn("request", fields...)
		default:
			logger.L().Info("request", fields...)
		}
	}
}
