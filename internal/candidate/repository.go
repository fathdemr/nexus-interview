package candidate

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

func (r *repository) Save(ctx context.Context, c *Candidate) error {
	if err := r.db.WithContext(ctx).Save(c).Error; err != nil {
		return fmt.Errorf("save candidate: %w", err)
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, id string) (Candidate, error) {
	var c Candidate
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Candidate{}, ErrNotFound
		}
		return Candidate{}, fmt.Errorf("query candidate %s: %w", id, err)
	}
	return c, nil
}

func (r *repository) FindByEmail(ctx context.Context, email string) (Candidate, error) {
	var c Candidate
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Candidate{}, ErrNotFound
		}
		return Candidate{}, fmt.Errorf("query candidate by email: %w", err)
	}
	return c, nil
}

func (r *repository) FindAll(ctx context.Context) ([]Candidate, error) {
	var candidates []Candidate
	if err := r.db.WithContext(ctx).Find(&candidates).Error; err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	return candidates, nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&Candidate{})
	if result.Error != nil {
		return fmt.Errorf("delete candidate %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
