package question

import (
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("question not found")
	ErrAlreadyExists = errors.New("question already exists")
)

// Service is the capability this module exposes to HTTP handlers and other modules.
type Service interface {
	CreateQuestion(ctx context.Context, req CreateRequest) (Question, error)
	FindByID(ctx context.Context, id string) (Question, error)
	ListByJobPosting(ctx context.Context, jobPostingID string) ([]Question, error)
	UpdateQuestion(ctx context.Context, id string, req UpdateRequest) (Question, error)
	DeleteQuestion(ctx context.Context, id string) error
}

// Repository is the data access contract owned by this module.
type Repository interface {
	Save(ctx context.Context, q *Question) error
	FindByID(ctx context.Context, id string) (Question, error)
	FindByJobPosting(ctx context.Context, jobPostingID string) ([]Question, error)
	Delete(ctx context.Context, id string) error
}

// CreateRequest carries the validated input for question creation.
type CreateRequest struct {
	JobPostingId string `json:"job_posting_id" binding:"required"`
	Text         string `json:"text"           binding:"required"`
	OrderIndex   int    `json:"order_index"`
}

// UpdateRequest carries the fields that may be updated on an existing question.
type UpdateRequest struct {
	Text       string `json:"text"`
	OrderIndex int    `json:"order_index"`
}
