package question

import (
	"errors"
	"net/http"
	"strings"

	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// Handler holds the question service and exposes HTTP endpoints.
type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("", h.createQuestion)
	rg.GET("/:id", h.getQuestion)
	rg.GET("/job-posting/:jobPostingId", h.listByJobPosting)
	rg.PUT("/:id", h.updateQuestion)
	rg.DELETE("/:id", h.deleteQuestion)
}

func (h *Handler) createQuestion(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), req)
		return
	}

	result, err := h.service.CreateQuestion(c.Request.Context(), req)
	if err != nil {
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to create question", "INTERNAL_ERROR"), req)
		return
	}

	httputil.GinJSON(c, http.StatusCreated,
		httputil.NewSuccessResponse("question created", "CREATED", result), req)
}

func (h *Handler) getQuestion(c *gin.Context) {
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
				httputil.NewErrorResponse("question not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to get question", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", result), nil)
}

func (h *Handler) listByJobPosting(c *gin.Context) {
	jobPostingId := strings.TrimSpace(c.Param("jobPostingId"))
	if jobPostingId == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid job posting id", "VALIDATION_ERROR"), nil)
		return
	}

	results, err := h.service.ListByJobPosting(c.Request.Context(), jobPostingId)
	if err != nil {
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to list questions", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", results), nil)
}

func (h *Handler) updateQuestion(c *gin.Context) {
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

	result, err := h.service.UpdateQuestion(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("question not found", "NOT_FOUND"), req)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to update question", "INTERNAL_ERROR"), req)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("question updated", "OK", result), req)
}

func (h *Handler) deleteQuestion(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	if err := h.service.DeleteQuestion(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("question not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to delete question", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("question deleted", "DELETED", nil), nil)
}
