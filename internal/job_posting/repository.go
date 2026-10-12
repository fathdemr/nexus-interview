package job_posting

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Save(ctx context.Context, jp *JobPosting) error {
	if err := r.db.WithContext(ctx).Save(jp).Error; err != nil {
		return fmt.Errorf("save job posting: %w", err)
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, id string) (JobPosting, error) {
	var jp JobPosting
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&jp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return JobPosting{}, ErrNotFound
		}
		return JobPosting{}, fmt.Errorf("query job posting %s: %w", id, err)
	}
	return jp, nil
}

func (r *repository) FindAll(ctx context.Context, filter ListFilter) ([]JobPosting, error) {
	var postings []JobPosting
	q := r.db.WithContext(ctx).Order("created_at DESC")
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if err := q.Find(&postings).Error; err != nil {
		return nil, fmt.Errorf("list job postings: %w", err)
	}
	return postings, nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&JobPosting{})
	if result.Error != nil {
		return fmt.Errorf("delete job posting %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repository) FindByShareToken(ctx context.Context, token string) (JobPosting, error) {
	var jp JobPosting
	err := r.db.WithContext(ctx).Where("share_token = ?", token).First(&jp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return JobPosting{}, ErrNotFound
		}
		return JobPosting{}, fmt.Errorf("query job posting by share token: %w", err)
	}
	return jp, nil
}
