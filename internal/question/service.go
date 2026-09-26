package question

import (
	"context"
	"fmt"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateQuestion(ctx context.Context, req CreateRequest) (Question, error) {
	q := Question{
		JobPostingId: req.JobPostingId,
		Text:         req.Text,
		OrderIndex:   req.OrderIndex,
	}
	if err := s.repo.Save(ctx, &q); err != nil {
		return Question{}, fmt.Errorf("create question: %w", err)
	}
	return q, nil
}

func (s *service) FindByID(ctx context.Context, id string) (Question, error) {
	q, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Question{}, fmt.Errorf("find question %s: %w", id, err)
	}
	return q, nil
}

func (s *service) ListByJobPosting(ctx context.Context, jobPostingID string) ([]Question, error) {
	questions, err := s.repo.FindByJobPosting(ctx, jobPostingID)
	if err != nil {
		return nil, fmt.Errorf("list questions for job posting %s: %w", jobPostingID, err)
	}
	return questions, nil
}

func (s *service) UpdateQuestion(ctx context.Context, id string, req UpdateRequest) (Question, error) {
	q, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Question{}, fmt.Errorf("update question %s: %w", id, err)
	}

	if req.Text != "" {
		q.Text = req.Text
	}
	if req.OrderIndex != 0 {
		q.OrderIndex = req.OrderIndex
	}

	if err := s.repo.Save(ctx, &q); err != nil {
		return Question{}, fmt.Errorf("save updated question %s: %w", id, err)
	}
	return q, nil
}

func (s *service) DeleteQuestion(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete question %s: %w", id, err)
	}
	return nil
}
