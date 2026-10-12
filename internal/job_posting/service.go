package job_posting

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

func (s *service) Create(ctx context.Context, req CreateRequest, actorID, actorName, actorIP string) (JobPosting, error) {
	jp := JobPosting{
		Title:       req.Title,
		Description: req.Description,
		Department:  req.Department,
		Location:    req.Location,
		Status:      StatusActive,
	}
	jp.SetCreatedUser(actorID, actorName, actorIP)

	if err := s.repo.Save(ctx, &jp); err != nil {
		return JobPosting{}, fmt.Errorf("create job posting: %w", err)
	}
	return jp, nil
}

func (s *service) FindByID(ctx context.Context, id string) (JobPosting, error) {
	jp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return JobPosting{}, fmt.Errorf("find job posting %s: %w", id, err)
	}
	return jp, nil
}

func (s *service) List(ctx context.Context, filter ListFilter) ([]JobPosting, error) {
	postings, err := s.repo.FindAll(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list job postings: %w", err)
	}
	return postings, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest, actorID, actorName, actorIP string) (JobPosting, error) {
	jp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return JobPosting{}, fmt.Errorf("update job posting %s: %w", id, err)
	}

	if req.Title != "" {
		jp.Title = req.Title
	}
	if req.Description != "" {
		jp.Description = req.Description
	}
	if req.Department != "" {
		jp.Department = req.Department
	}
	if req.Location != "" {
		jp.Location = req.Location
	}

	jp.SetUpdatedUser(actorID, actorName, actorIP)

	if err := s.repo.Save(ctx, &jp); err != nil {
		return JobPosting{}, fmt.Errorf("save job posting %s: %w", id, err)
	}
	return jp, nil
}

func (s *service) SetStatus(ctx context.Context, id string, status JobPostingStatus, actorID, actorName, actorIP string) (JobPosting, error) {
	if status != StatusActive && status != StatusPassive {
		return JobPosting{}, ErrInvalidStatus
	}

	jp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return JobPosting{}, fmt.Errorf("set status job posting %s: %w", id, err)
	}

	jp.Status = status
	jp.SetUpdatedUser(actorID, actorName, actorIP)

	if err := s.repo.Save(ctx, &jp); err != nil {
		return JobPosting{}, fmt.Errorf("save job posting %s: %w", id, err)
	}
	return jp, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete job posting %s: %w", id, err)
	}
	return nil
}

func (s *service) FindByShareToken(ctx context.Context, token string) (JobPosting, error) {
	jp, err := s.repo.FindByShareToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return JobPosting{}, ErrNotFound
		}
		return JobPosting{}, fmt.Errorf("find job posting by token: %w", err)
	}
	// Only active postings are accessible via share link
	if jp.Status != StatusActive {
		return JobPosting{}, ErrNotFound
	}
	return jp, nil
}
