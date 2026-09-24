package candidate

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// Handler holds the candidate service and exposes HTTP endpoints.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("", h.createCandidate)
	rg.GET("", h.listCandidates)
	rg.GET("/:id", h.getCandidate)
	rg.PUT("/:id", h.updateCandidate)
	rg.DELETE("/:id", h.deleteCandidate)
}

func (h *Handler) createCandidate(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), req)
		return
	}

	result, err := h.service.CreateCandidate(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			httputil.GinJSON(c, http.StatusConflict,
				httputil.NewErrorResponse("candidate already exists", "ALREADY_EXISTS"), req)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to create candidate", "INTERNAL_ERROR"), req)
		return
	}

	httputil.GinJSON(c, http.StatusCreated,
		httputil.NewSuccessResponse("candidate created", "CREATED", result), req)
}

func (h *Handler) listCandidates(c *gin.Context) {
	candidates, err := h.service.ListCandidates(c.Request.Context())
	if err != nil {
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to list candidates", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", candidates), nil)
}

func (h *Handler) getCandidate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	result, err := h.service.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("candidate not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to get candidate", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", result), nil)
}

func (h *Handler) updateCandidate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), req)
		return
	}

	result, err := h.service.UpdateCandidate(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("candidate not found", "NOT_FOUND"), req)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to update candidate", "INTERNAL_ERROR"), req)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("candidate updated", "OK", result), req)
}

func (h *Handler) deleteCandidate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	if err := h.service.DeleteCandidate(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("candidate not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to delete candidate", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("candidate deleted", "DELETED", nil), nil)
}
