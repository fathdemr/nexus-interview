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

// createQuestion adds an interview question to a job posting.
//
// @Summary      Create question
// @Description  Creates a question bound to the given `job_posting_id`. `order_index` controls the position of the
// @Description  question inside the interview (lower first, default 0). The generated UUID is returned in `data.id`.
// @Tags         Questions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRequest  true  "Question to create"
// @Success      201   {object}  httputil.BaseServiceResponse{data=Question}  "CREATED"
// @Failure      400   {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — missing job_posting_id or text"
// @Failure      401   {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      500   {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /questions [post]
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

// getQuestion returns a single question by ID.
//
// @Summary      Get question
// @Description  Fetches one question by its UUID. Soft-deleted questions are treated as not found.
// @Tags         Questions
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Question UUID"  format(uuid)
// @Success      200  {object}  httputil.BaseServiceResponse{data=Question}  "OK"
// @Failure      400  {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — empty id"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404  {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /questions/{id} [get]
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

// listByJobPosting returns all questions for one job posting.
//
// @Summary      List questions of a job posting
// @Description  Returns every non-deleted question attached to the given job posting, ordered by `order_index`.
// @Description  An unknown job posting yields an empty list, not 404.
// @Tags         Questions
// @Security     BearerAuth
// @Produce      json
// @Param        jobPostingId  path      string  true  "Job posting UUID"  format(uuid)
// @Success      200  {object}  httputil.BaseServiceResponse{data=[]Question}  "OK"
// @Failure      400  {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — empty job posting id"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /questions/job-posting/{jobPostingId} [get]
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

// updateQuestion modifies the text and/or ordering of a question.
//
// @Summary      Update question
// @Description  Updates `text` and `order_index` of an existing question. The owning job posting cannot be changed;
// @Description  delete and recreate the question to move it. Returns the full updated record.
// @Tags         Questions
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string         true  "Question UUID"  format(uuid)
// @Param        body  body      UpdateRequest  true  "Fields to update"
// @Success      200   {object}  httputil.BaseServiceResponse{data=Question}  "OK"
// @Failure      400   {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — empty id or malformed body"
// @Failure      401   {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404   {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500   {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /questions/{id} [put]
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

// deleteQuestion soft-deletes a question.
//
// @Summary      Delete question
// @Description  Marks the question as deleted (soft delete). It no longer appears in job posting listings.
// @Description  Deleting an already-deleted or unknown id yields 404.
// @Tags         Questions
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Question UUID"  format(uuid)
// @Success      200  {object}  httputil.BaseServiceResponse  "DELETED"
// @Failure      400  {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — empty id"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404  {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /questions/{id} [delete]
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
