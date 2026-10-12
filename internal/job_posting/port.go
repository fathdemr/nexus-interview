package job_posting

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("job posting not found")
	ErrAlreadyExists = errors.New("job posting already exists")
	ErrInvalidStatus = errors.New("invalid status: must be active or passive")
)

// Service defines all business operations on job postings.
type Service interface {
	Create(ctx context.Context, req CreateRequest, actorID, actorName, actorIP string) (JobPosting, error)
	FindByID(ctx context.Context, id string) (JobPosting, error)
	List(ctx context.Context, filter ListFilter) ([]JobPosting, error)
	Update(ctx context.Context, id string, req UpdateRequest, actorID, actorName, actorIP string) (JobPosting, error)
	SetStatus(ctx context.Context, id string, status JobPostingStatus, actorID, actorName, actorIP string) (JobPosting, error)
	Delete(ctx context.Context, id string) error
	FindByShareToken(ctx context.Context, token string) (JobPosting, error)
}

// Repository is the data access contract for the job_posting module.
type Repository interface {
	Save(ctx context.Context, jp *JobPosting) error
	FindByID(ctx context.Context, id string) (JobPosting, error)
	FindAll(ctx context.Context, filter ListFilter) ([]JobPosting, error)
	Delete(ctx context.Context, id string) error
	FindByShareToken(ctx context.Context, token string) (JobPosting, error)
}

// CreateRequest is the validated payload for POST /job-postings.
type CreateRequest struct {
	Title       string `json:"title"       binding:"required,min=3,max=255"`
	Description string `json:"description" binding:"required"`
	Department  string `json:"department"  binding:"max=100"`
	Location    string `json:"location"    binding:"max=100"`
}

// UpdateRequest is the validated payload for PUT /job-postings/:id.
// All fields are optional — only non-empty values are applied.
type UpdateRequest struct {
	Title       string `json:"title"       binding:"omitempty,min=3,max=255"`
	Description string `json:"description"`
	Department  string `json:"department"  binding:"omitempty,max=100"`
	Location    string `json:"location"    binding:"omitempty,max=100"`
}

// SetStatusRequest is the body for PATCH /job-postings/:id/status.
type SetStatusRequest struct {
	Status JobPostingStatus `json:"status" binding:"required"`
}

// ListFilter controls optional server-side filtering on the list endpoint.
type ListFilter struct {
	Status string // "active" | "passive" | "" (all)
}

// JobPostingResponse is the API representation. Adds a share URL field.
type JobPostingResponse struct {
	JobPosting
	ShareURL string `json:"share_url"`
}
