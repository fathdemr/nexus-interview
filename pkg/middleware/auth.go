package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fathdemr/nexus-interview/internal/auth"
	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// CheckToken enforces authentication on a route group.
// Aborts with 401 if the token is missing, invalid, or expired.
// Sets the Actor on the context for downstream handlers.
func CheckToken(validator auth.TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		setCORSHeaders(c)

		raw := extractBearerToken(c)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				httputil.NewErrorResponse("missing token", "UNAUTHORIZED"))
			return
		}

		claims, err := validator.ValidateToken(c.Request.Context(), raw)
		if err != nil {
			code := "UNAUTHORIZED"
			if errors.Is(err, auth.ErrTokenExpired) {
				code = "TOKEN_EXPIRED"
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				httputil.NewErrorResponse("unauthorized", code))
			return
		}

		httputil.SetActor(c, httputil.Actor{
			Id:       claims.UserID,
			FullName: claims.FullName,
			Email:    claims.Email,
			Role:     claims.Role,
			ClientIP: c.ClientIP(),
		})

		c.Next()
	}
}

// CheckTokenWithoutAbort attempts to populate the Actor from the token but
// never aborts. Routes that serve both authenticated and anonymous users
// should use this middleware and check Actor.Id themselves.
func CheckTokenWithoutAbort(validator auth.TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		setCORSHeaders(c)

		raw := extractBearerToken(c)
		if raw == "" {
			c.Next()
			return
		}

		claims, err := validator.ValidateToken(c.Request.Context(), raw)
		if err != nil {
			c.Next()
			return
		}

		httputil.SetActor(c, httputil.Actor{
			Id:       claims.UserID,
			FullName: claims.FullName,
			Email:    claims.Email,
			Role:     claims.Role,
			ClientIP: c.ClientIP(),
		})

		c.Next()
	}
}

// extractBearerToken reads the token from the Authorization header first,
// then falls back to the "access_token" HttpOnly cookie.
func extractBearerToken(c *gin.Context) string {
	raw := c.GetHeader("Authorization")
	raw = strings.TrimPrefix(raw, "Bearer ")
	raw = strings.TrimPrefix(raw, "bearer ")
	raw = strings.TrimSpace(raw)
	if raw != "" {
		return raw
	}
	// Fallback to HttpOnly cookie set by auth.Handler.SetTokenCookies
	cookie, _ := c.Cookie("access_token")
	return strings.TrimSpace(cookie)
}

// setCORSHeaders adds permissive CORS headers for all responses.
// Tighten origin policy before moving to production.
func setCORSHeaders(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	if c.Request.Method == http.MethodOptions {
		if v := c.GetHeader("Access-Control-Request-Headers"); v != "" {
			c.Header("Access-Control-Allow-Headers", v)
		}
		c.AbortWithStatus(http.StatusNoContent)
	}
}
