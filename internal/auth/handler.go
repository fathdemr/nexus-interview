package auth

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/fathdemr/nexus-interview/pkg/ratelimit"
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

// Handler exposes auth endpoints: register, login, refresh and logout.
type Handler struct {
	service Service
	cfg     HandlerConfig
	limiter *ratelimit.Limiter
}

func NewHandler(service Service, cfg HandlerConfig, limiter *ratelimit.Limiter) *Handler {
	return &Handler{service: service, cfg: cfg, limiter: limiter}
}

// RegisterRoutes attaches public auth routes to the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/register", h.limiter.Middleware("register"), h.register)
	rg.POST("/login", h.limiter.Middleware("login"), h.login)
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

// register creates a new user account and returns an access/refresh token pair as cookies.
//
// @Summary      Register
// @Description  Creates a new user with the specified role (admin or candidate).
// @Description  On success, sets `access_token` and `refresh_token` as HttpOnly cookies.
// @Description  Returns 409 if the email is already registered.
// @Description  Returns 429 after 5 consecutive failed attempts from the same IP (locked for 1 minute).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest               true  "Registration payload"
// @Success      201   {object}  httputil.BaseServiceResponse        "account created — cookies set"
// @Failure      400   {object}  httputil.BaseServiceResponse        "VALIDATION_ERROR"
// @Failure      409   {object}  httputil.BaseServiceResponse        "CONFLICT — email already registered"
// @Failure      429   {object}  httputil.BaseServiceResponse        "RATE_LIMITED — too many failed attempts"
// @Failure      500   {object}  httputil.BaseServiceResponse        "INTERNAL_ERROR"
// @Router       /auth/register [post]
func (h *Handler) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.limiter.RecordFailure("register", c.ClientIP())
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), nil)
		return
	}

	pair, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		locked, retryAfter := h.limiter.RecordFailure("register", c.ClientIP())
		if locked {
			secs := int(retryAfter.Seconds())
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", secs))
			httputil.GinJSON(c, http.StatusTooManyRequests,
				httputil.NewErrorResponse(
					fmt.Sprintf("too many failed attempts, try again in %ds", secs),
					"RATE_LIMITED",
				), nil)
			return
		}

		switch {
		case errors.Is(err, ErrAlreadyExists):
			httputil.GinJSON(c, http.StatusConflict,
				httputil.NewErrorResponse("email already registered", "CONFLICT"), nil)
		case errors.Is(err, ErrInvalidRole):
			httputil.GinJSON(c, http.StatusBadRequest,
				httputil.NewErrorResponse("role must be admin or candidate", "VALIDATION_ERROR"), nil)
		default:
			httputil.GinJSON(c, http.StatusInternalServerError,
				httputil.NewErrorResponse("registration failed", "INTERNAL_ERROR"), nil)
		}
		return
	}

	h.limiter.Reset("register", c.ClientIP())
	h.SetTokenCookies(c, pair)
	httputil.GinJSON(c, http.StatusCreated,
		httputil.NewSuccessResponse("account created", "CREATED", nil), nil)
}

// login authenticates a user with email and password and returns a token pair as cookies.
//
// @Summary      Login
// @Description  Validates the email/password pair and, on success, issues a new access/refresh token pair
// @Description  written as HttpOnly cookies. Returns a generic 401 for both unknown email and wrong password
// @Description  to prevent user enumeration.
// @Description  Returns 429 after 5 consecutive failed attempts from the same IP (locked for 1 minute).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest                  true  "Login credentials"
// @Success      200   {object}  httputil.BaseServiceResponse        "authenticated — cookies set"
// @Failure      400   {object}  httputil.BaseServiceResponse        "VALIDATION_ERROR"
// @Failure      401   {object}  httputil.BaseServiceResponse        "UNAUTHORIZED — invalid credentials"
// @Failure      429   {object}  httputil.BaseServiceResponse        "RATE_LIMITED — too many failed attempts"
// @Failure      500   {object}  httputil.BaseServiceResponse        "INTERNAL_ERROR"
// @Router       /auth/login [post]
func (h *Handler) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.limiter.RecordFailure("login", c.ClientIP())
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), nil)
		return
	}

	pair, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		locked, retryAfter := h.limiter.RecordFailure("login", c.ClientIP())
		if locked {
			secs := int(retryAfter.Seconds())
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", fmt.Sprintf("%d", secs))
			httputil.GinJSON(c, http.StatusTooManyRequests,
				httputil.NewErrorResponse(
					fmt.Sprintf("too many failed attempts, try again in %ds", secs),
					"RATE_LIMITED",
				), nil)
			return
		}

		// Return the same message for ErrNotFound and ErrInvalidPassword to prevent
		// user enumeration — an attacker must not be able to tell whether an email exists.
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrInvalidPassword) {
			httputil.GinJSON(c, http.StatusUnauthorized,
				httputil.NewErrorResponse("invalid email or password", "UNAUTHORIZED"), nil)
			return
		}

		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("login failed", "INTERNAL_ERROR"), nil)
		return
	}

	h.limiter.Reset("login", c.ClientIP())
	h.SetTokenCookies(c, pair)
	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("authenticated", "OK", nil), nil)
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
