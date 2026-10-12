package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/fathdemr/nexus-interview/pkg/model"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, ErrNotFound
		}
		return model.User{}, fmt.Errorf("query user by email: %w", err)
	}
	return user, nil
}

func (r *repository) IsExists(ctx context.Context, email string) bool {
	var count int64
	return r.db.Table("users").WithContext(ctx).Where("email = ?", email).Count(&count).Error == nil && count > 0
}

func (r *repository) Create(ctx context.Context, user model.User) (model.User, error) {
	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return model.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}
