package question

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

func (r *repository) Save(ctx context.Context, q *Question) error {
	if err := r.db.WithContext(ctx).Save(q).Error; err != nil {
		return fmt.Errorf("save question: %w", err)
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, id string) (Question, error) {
	var q Question
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&q).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Question{}, ErrNotFound
		}
		return Question{}, fmt.Errorf("query question %s: %w", id, err)
	}
	return q, nil
}

func (r *repository) FindByJobPosting(ctx context.Context, jobPostingID string) ([]Question, error) {
	var questions []Question
	if err := r.db.WithContext(ctx).
		Where("job_posting_id = ?", jobPostingID).
		Order("order_index asc").
		Find(&questions).Error; err != nil {
		return nil, fmt.Errorf("list questions for job posting %s: %w", jobPostingID, err)
	}
	return questions, nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&Question{})
	if result.Error != nil {
		return fmt.Errorf("delete question %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
