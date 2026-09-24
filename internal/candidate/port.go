package candidate

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("candidate not found")
	ErrAlreadyExists = errors.New("candidate already exists")
)

// Service is the capability this module exposes to HTTP handlers and other modules.
type Service interface {
	CreateCandidate(ctx context.Context, req CreateRequest) (Candidate, error)
	FindByID(ctx context.Context, id string) (Candidate, error)
	ListCandidates(ctx context.Context) ([]Candidate, error)
	UpdateCandidate(ctx context.Context, id string, req UpdateRequest) (Candidate, error)
	DeleteCandidate(ctx context.Context, id string) error
}

// Repository is the data access contract owned by this module.
type Repository interface {
	Save(ctx context.Context, c *Candidate) error
	FindByID(ctx context.Context, id string) (Candidate, error)
	FindByEmail(ctx context.Context, email string) (Candidate, error)
	FindAll(ctx context.Context) ([]Candidate, error)
	Delete(ctx context.Context, id string) error
}

// CreateRequest carries the validated input for candidate creation.
type CreateRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email"     binding:"required,email"`
}

// UpdateRequest carries the fields that may be updated on an existing candidate.
type UpdateRequest struct {
	FullName string `json:"full_name"`
}
