package model

import (
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

// UserRole represents the role assigned to a user in the system.
type UserRole string

const (
	// RoleAdmin grants access to admin-only endpoints (user management, reporting, etc.).
	RoleAdmin UserRole = "admin"
	// RoleCandidate is assigned to job applicants invited to take interviews.
	RoleCandidate UserRole = "candidate"
)

// User is the central identity record shared across all roles.
// Authentication happens through this model; role-specific data lives
// in separate tables (e.g. Candidate) linked by email or user ID.
type User struct {
	Id string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	FirstName string   `gorm:"type:varchar(255);not null" json:"first_name"`
	LastName  string   `gorm:"type:varchar(255);not null" json:"last_name"`
	Email     string   `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password  string   `gorm:"type:varchar(255);not null" json:"-"` // bcrypt hash, never serialised
	Role      UserRole `gorm:"type:varchar(50);not null" json:"role"`

	BaseRecordFields
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Id == "" {
		u.Id = uuid.Must(uuid.NewV7()).String()
	}
	return nil
}
