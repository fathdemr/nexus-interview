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

// createCandidate registers a new job applicant.
//
// @Summary      Create candidate
// @Description  Creates a candidate record from full name and e-mail. E-mail is unique across candidates;
// @Description  a duplicate yields 409. The generated UUID is returned in `data.id`.
// @Tags         Candidates
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRequest  true  "Candidate to create"
// @Success      201   {object}  httputil.BaseServiceResponse{data=Candidate}  "CREATED"
// @Failure      400   {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — missing full_name or malformed email"
// @Failure      401   {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      409   {object}  httputil.BaseServiceResponse  "ALREADY_EXISTS — a candidate with this email already exists"
// @Failure      500   {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /candidates [post]
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

// listCandidates returns every non-deleted candidate.
//
// @Summary      List candidates
// @Description  Returns all candidates that have not been soft-deleted. No pagination or filtering is applied yet.
// @Tags         Candidates
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  httputil.BaseServiceResponse{data=[]Candidate}  "OK"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /candidates [get]
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

// getCandidate returns a single candidate by ID.
//
// @Summary      Get candidate
// @Description  Fetches one candidate by its UUID. Soft-deleted candidates are treated as not found.
// @Tags         Candidates
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Candidate UUID"  format(uuid)
// @Success      200  {object}  httputil.BaseServiceResponse{data=Candidate}  "OK"
// @Failure      400  {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — empty id"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404  {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /candidates/{id} [get]
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

// updateCandidate modifies mutable fields of an existing candidate.
//
// @Summary      Update candidate
// @Description  Updates the candidate's display name. E-mail is the login identifier and cannot be changed here.
// @Description  Returns the full updated record.
// @Tags         Candidates
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string         true  "Candidate UUID"  format(uuid)
// @Param        body  body      UpdateRequest  true  "Fields to update"
// @Success      200   {object}  httputil.BaseServiceResponse{data=Candidate}  "OK"
// @Failure      400   {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — empty id or malformed body"
// @Failure      401   {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404   {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500   {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /candidates/{id} [put]
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

// deleteCandidate soft-deletes a candidate.
//
// @Summary      Delete candidate
// @Description  Marks the candidate as deleted (soft delete). The record is retained in the database but
// @Description  disappears from list and get endpoints. Deleting an already-deleted or unknown id yields 404.
// @Tags         Candidates
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Candidate UUID"  format(uuid)
// @Success      200  {object}  httputil.BaseServiceResponse  "DELETED"
// @Failure      400  {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR — empty id"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404  {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /candidates/{id} [delete]
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
