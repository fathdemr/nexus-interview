package candidate

import (
	"context"
	"errors"
	"fmt"
)

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateCandidate(ctx context.Context, req CreateRequest) (Candidate, error) {
	_, err := s.repo.FindByEmail(ctx, req.Email)
	if err == nil {
		return Candidate{}, ErrAlreadyExists
	}
	if !errors.Is(err, ErrNotFound) {
		return Candidate{}, fmt.Errorf("check existing candidate: %w", err)
	}

	c := Candidate{
		FullName: req.FullName,
		Email:    req.Email,
	}
	if err := s.repo.Save(ctx, &c); err != nil {
		return Candidate{}, fmt.Errorf("create candidate: %w", err)
	}
	return c, nil
}

func (s *service) FindByID(ctx context.Context, id string) (Candidate, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Candidate{}, fmt.Errorf("find candidate %s: %w", id, err)
	}
	return c, nil
}

func (s *service) ListCandidates(ctx context.Context) ([]Candidate, error) {
	candidates, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	return candidates, nil
}

func (s *service) UpdateCandidate(ctx context.Context, id string, req UpdateRequest) (Candidate, error) {
	c, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return Candidate{}, fmt.Errorf("update candidate %s: %w", id, err)
	}

	if req.FullName != "" {
		c.FullName = req.FullName
	}

	if err := s.repo.Save(ctx, &c); err != nil {
		return Candidate{}, fmt.Errorf("save updated candidate %s: %w", id, err)
	}
	return c, nil
}

func (s *service) DeleteCandidate(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete candidate %s: %w", id, err)
	}
	return nil
}
