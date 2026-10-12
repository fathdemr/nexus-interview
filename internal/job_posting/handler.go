package job_posting

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fathdemr/nexus-interview/pkg/httputil"
	"github.com/gin-gonic/gin"
)

// Handler exposes job posting endpoints.
// baseURL is used to build the shareable link returned in responses.
// Example: "https://nexus-interview.com"
type Handler struct {
	service Service
	baseURL string
}

func NewHandler(service Service, baseURL string) *Handler {
	return &Handler{service: service, baseURL: baseURL}
}

// RegisterRoutes attaches all job posting routes to the given (protected) router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("", h.create)
	rg.GET("", h.list)
	rg.GET("/:id", h.getByID)
	rg.PUT("/:id", h.update)
	rg.PATCH("/:id/status", h.setStatus)
	rg.DELETE("/:id", h.delete)
}

// RegisterPublicRoutes attaches routes accessible without authentication (share link lookup).
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/share/:token", h.getByShareToken)
}

// create adds a new job posting.
//
// @Summary      Create job posting
// @Description  Creates a new job posting with status=active. A unique share token is generated automatically.
// @Description  The shareable public URL is returned in `data.share_url`.
// @Tags         Job Postings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      CreateRequest  true  "Job posting fields"
// @Success      201   {object}  httputil.BaseServiceResponse{data=JobPostingResponse}  "CREATED"
// @Failure      400   {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR"
// @Failure      401   {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      500   {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /job-postings [post]
func (h *Handler) create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), nil)
		return
	}

	actor := httputil.GetActor(c)
	jp, err := h.service.Create(c.Request.Context(), req, actor.Id, actor.FullName, actor.ClientIP)
	if err != nil {
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to create job posting", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusCreated,
		httputil.NewSuccessResponse("job posting created", "CREATED", h.toResponse(jp)), nil)
}

// list returns job postings, optionally filtered by status.
//
// @Summary      List job postings
// @Description  Returns all job postings ordered by creation date (newest first).
// @Description  Pass `?status=active` or `?status=passive` to filter. Omit for all.
// @Tags         Job Postings
// @Security     BearerAuth
// @Produce      json
// @Param        status  query     string  false  "Filter by status"  Enums(active, passive)
// @Success      200     {object}  httputil.BaseServiceResponse{data=[]JobPostingResponse}  "OK"
// @Failure      401     {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      500     {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /job-postings [get]
func (h *Handler) list(c *gin.Context) {
	filter := ListFilter{Status: strings.TrimSpace(c.Query("status"))}

	postings, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to list job postings", "INTERNAL_ERROR"), nil)
		return
	}

	resp := make([]JobPostingResponse, len(postings))
	for i, jp := range postings {
		resp[i] = h.toResponse(jp)
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", resp), nil)
}

// getByID returns a single job posting by its UUID.
//
// @Summary      Get job posting
// @Description  Fetches one job posting by its UUID. Soft-deleted postings are treated as not found.
// @Tags         Job Postings
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Job Posting UUID"  format(uuid)
// @Success      200  {object}  httputil.BaseServiceResponse{data=JobPostingResponse}  "OK"
// @Failure      400  {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404  {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /job-postings/{id} [get]
func (h *Handler) getByID(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	jp, err := h.service.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("job posting not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to get job posting", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", h.toResponse(jp)), nil)
}

// update modifies an existing job posting's content fields.
//
// @Summary      Update job posting
// @Description  Updates title, description, department and/or location. All fields are optional;
// @Description  only provided non-empty values are applied. Status is changed via PATCH /:id/status.
// @Tags         Job Postings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string        true  "Job Posting UUID"  format(uuid)
// @Param        body  body      UpdateRequest true  "Fields to update"
// @Success      200   {object}  httputil.BaseServiceResponse{data=JobPostingResponse}  "OK"
// @Failure      400   {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR"
// @Failure      401   {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404   {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500   {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /job-postings/{id} [put]
func (h *Handler) update(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), nil)
		return
	}

	actor := httputil.GetActor(c)
	jp, err := h.service.Update(c.Request.Context(), id, req, actor.Id, actor.FullName, actor.ClientIP)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("job posting not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to update job posting", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("job posting updated", "OK", h.toResponse(jp)), nil)
}

// setStatus activates or deactivates a job posting.
//
// @Summary      Set job posting status
// @Description  Changes the status of a job posting to `active` or `passive`.
// @Description  Passive postings are hidden from the public share link.
// @Tags         Job Postings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string           true  "Job Posting UUID"  format(uuid)
// @Param        body  body      SetStatusRequest true  "New status"
// @Success      200   {object}  httputil.BaseServiceResponse{data=JobPostingResponse}  "OK"
// @Failure      400   {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR"
// @Failure      401   {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404   {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500   {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /job-postings/{id}/status [patch]
func (h *Handler) setStatus(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	var req SetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse(err.Error(), "VALIDATION_ERROR"), nil)
		return
	}

	actor := httputil.GetActor(c)
	jp, err := h.service.SetStatus(c.Request.Context(), id, req.Status, actor.Id, actor.FullName, actor.ClientIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("job posting not found", "NOT_FOUND"), nil)
		case errors.Is(err, ErrInvalidStatus):
			httputil.GinJSON(c, http.StatusBadRequest,
				httputil.NewErrorResponse("status must be active or passive", "VALIDATION_ERROR"), nil)
		default:
			httputil.GinJSON(c, http.StatusInternalServerError,
				httputil.NewErrorResponse("failed to set status", "INTERNAL_ERROR"), nil)
		}
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("status updated", "OK", h.toResponse(jp)), nil)
}

// delete permanently removes a job posting (hard delete via soft delete column).
//
// @Summary      Delete job posting
// @Description  Soft-deletes a job posting. The record is retained in the database but no longer
// @Description  appears in any listing or detail endpoint, and its share link stops working.
// @Tags         Job Postings
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Job Posting UUID"  format(uuid)
// @Success      200  {object}  httputil.BaseServiceResponse  "DELETED"
// @Failure      400  {object}  httputil.BaseServiceResponse  "VALIDATION_ERROR"
// @Failure      401  {object}  httputil.BaseServiceResponse  "UNAUTHORIZED"
// @Failure      404  {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500  {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /job-postings/{id} [delete]
func (h *Handler) delete(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		httputil.GinJSON(c, http.StatusBadRequest,
			httputil.NewErrorResponse("invalid id", "VALIDATION_ERROR"), nil)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("job posting not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to delete job posting", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("job posting deleted", "DELETED", nil), nil)
}

// getByShareToken is the public endpoint that resolves a share link.
//
// @Summary      Get job posting by share token
// @Description  Looks up an active job posting via its share token. This endpoint is public — no auth required.
// @Description  Passive or deleted postings return 404 to prevent information disclosure.
// @Tags         Job Postings
// @Produce      json
// @Param        token  path      string  true  "Share token"
// @Success      200    {object}  httputil.BaseServiceResponse{data=JobPostingResponse}  "OK"
// @Failure      404    {object}  httputil.BaseServiceResponse  "NOT_FOUND"
// @Failure      500    {object}  httputil.BaseServiceResponse  "INTERNAL_ERROR"
// @Router       /job-postings/share/{token} [get]
func (h *Handler) getByShareToken(c *gin.Context) {
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		httputil.GinJSON(c, http.StatusNotFound,
			httputil.NewErrorResponse("not found", "NOT_FOUND"), nil)
		return
	}

	jp, err := h.service.FindByShareToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.GinJSON(c, http.StatusNotFound,
				httputil.NewErrorResponse("not found", "NOT_FOUND"), nil)
			return
		}
		httputil.GinJSON(c, http.StatusInternalServerError,
			httputil.NewErrorResponse("failed to get job posting", "INTERNAL_ERROR"), nil)
		return
	}

	httputil.GinJSON(c, http.StatusOK,
		httputil.NewSuccessResponse("ok", "OK", h.toResponse(jp)), nil)
}

// toResponse converts a JobPosting to its API response form, adding the share URL.
func (h *Handler) toResponse(jp JobPosting) JobPostingResponse {
	return JobPostingResponse{
		JobPosting: jp,
		ShareURL:   fmt.Sprintf("%s/apply/%s", strings.TrimRight(h.baseURL, "/"), jp.ShareToken),
	}
}
