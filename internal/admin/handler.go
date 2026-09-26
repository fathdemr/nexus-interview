package admin

import (
	"net/http"

	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// Handler holds the admin service and exposes HTTP endpoints.
type Handler struct {
	service Service
}

// NewHandler constructs an admin Handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes attaches all admin routes to the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/healthcheck", h.healthcheck)
}

// healthcheck verifies the service and its dependencies are operational.
//
// @Summary      Dependency healthcheck
// @Description  Runs the admin service healthcheck (database and cache connectivity). Intended for authenticated
// @Description  operators and readiness probes that hold a token; use the public /healthcheck for plain liveness.
// @Tags         Admin
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  httputil.BaseServiceResponse  "OK — all dependencies reachable"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR — one or more dependencies unhealthy"
// @Router       /admin/healthcheck [get]
func (h *Handler) healthcheck(c *gin.Context) {
	if err := h.service.Healthcheck(c.Request.Context()); err != nil {
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("service unhealthy", "INTERNAL_ERROR"), nil)
		return
	}
	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", nil), nil)
}
