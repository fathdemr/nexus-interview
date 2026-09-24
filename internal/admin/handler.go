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

func (h *Handler) healthcheck(c *gin.Context) {
	if err := h.service.Healthcheck(c.Request.Context()); err != nil {
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("service unhealthy", "INTERNAL_ERROR"), nil)
		return
	}
	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", nil), nil)
}
