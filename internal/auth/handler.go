package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/gin-gonic/gin"
)

const (
	accessTokenCookie  = "access_token"
	refreshTokenCookie = "refresh_token"
)

// HandlerConfig carries cookie settings injected from application config.
type HandlerConfig struct {
	// CookieDomain is the domain attribute on both cookies.
	// Example: "nexus-interview.com" — leave empty for localhost.
	CookieDomain string

	// CookieSecure controls the Secure flag. Set true in production (HTTPS only).
	CookieSecure bool

	// RefreshTokenExpirationDays is used to set the refresh cookie Max-Age.
	RefreshTokenExpirationDays int
}

// Handler exposes auth endpoints: refresh and logout.
// Token issuance (login) is the responsibility of the module that owns user credentials
// (e.g. admin or candidate) — they call IssueTokenPair and delegate cookie writing here.
type Handler struct {
	service Service
	cfg     HandlerConfig
}

func NewHandler(service Service, cfg HandlerConfig) *Handler {
	return &Handler{service: service, cfg: cfg}
}

// RegisterRoutes attaches the refresh route to the given router group.
// Logout is registered separately on a protected group (requires valid token).
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/refresh", h.refresh)
}

// RegisterProtectedRoutes attaches routes that require authentication.
func (h *Handler) RegisterProtectedRoutes(rg *gin.RouterGroup) {
	rg.POST("/logout", h.logout)
}

// SetTokenCookies writes the access and refresh tokens as HttpOnly cookies.
// Call this from any handler that issues a new token pair (login, refresh).
func (h *Handler) SetTokenCookies(c *gin.Context, pair TokenPair) {
	accessMaxAge := int(time.Until(pair.AccessTokenExpiresAt).Seconds())
	refreshMaxAge := h.cfg.RefreshTokenExpirationDays * 24 * 60 * 60

	// HttpOnly=true — JavaScript cannot read this cookie, mitigating XSS token theft.
	// SameSite=Strict — cookie is not sent on cross-site requests, mitigating CSRF.
	c.SetCookie(accessTokenCookie, pair.AccessToken, accessMaxAge,
		"/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)

	// Refresh token is scoped to /api/v1/auth to minimise exposure surface.
	c.SetCookie(refreshTokenCookie, pair.RefreshToken, refreshMaxAge,
		"/api/v1/auth", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}

// ClearTokenCookies removes both auth cookies from the browser.
func (h *Handler) ClearTokenCookies(c *gin.Context) {
	c.SetCookie(accessTokenCookie, "", -1, "/", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
	c.SetCookie(refreshTokenCookie, "", -1, "/api/v1/auth", h.cfg.CookieDomain, h.cfg.CookieSecure, true)
}

// refresh rotates the token pair using the HttpOnly refresh cookie.
//
// @Summary      Refresh access token
// @Description  Reads the `refresh_token` HttpOnly cookie (scoped to /api/v1/auth), validates it against the token store,
// @Description  and issues a brand-new access/refresh pair. Both tokens are written back as HttpOnly cookies;
// @Description  the response body carries no token. If the refresh token is unknown or expired the cookies are cleared
// @Description  and the client must log in again.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  httputil.BaseServiceResponse  "tokens refreshed — new cookies set"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED — refresh cookie missing, invalid or expired"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /auth/refresh [post]
func (h *Handler) refresh(c *gin.Context) {
	refreshToken, err := c.Cookie(refreshTokenCookie)
	if err != nil || refreshToken == "" {
		httputil.GinJSON(c, http.StatusUnauthorized,
			httputil.NewErrorResponse("refresh token missing", "UNAUTHORIZED"), nil)
		return
	}

	pair, err := h.service.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, ErrRefreshNotFound) {
			h.ClearTokenCookies(c)
			httputil.GinJSON(c, http.StatusUnauthorized,
				httputil.NewErrorResponse("refresh token invalid or expired", "UNAUTHORIZED"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("token refresh failed", "INTERNAL_ERROR"), nil)
		return
	}

	h.SetTokenCookies(c, pair)
	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("tokens refreshed", "OK", nil), nil)
}

// logout revokes the caller's refresh token and clears both auth cookies.
//
// @Summary      Logout
// @Description  Invalidates the refresh token stored for the authenticated actor so it can no longer be used to mint
// @Description  new access tokens, then expires the `access_token` and `refresh_token` cookies in the browser.
// @Description  The current access token stays valid until its natural expiry.
// @Tags         Auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  httputil.BaseServiceResponse  "logged out — cookies cleared"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED — missing or invalid access token"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /auth/logout [post]
func (h *Handler) logout(c *gin.Context) {
	actor := httputil.GetActor(c)

	if actor.Id != "" {
		if err := h.service.Logout(c.Request.Context(), actor.Id); err != nil {
			httputil.GinJSON(c, http.StatusInternalServerError,
				httputil.NewErrorResponse("logout failed", "INTERNAL_ERROR"), nil)
			return
		}
	}

	h.ClearTokenCookies(c)
	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("logged out", "OK", nil), nil)
}
