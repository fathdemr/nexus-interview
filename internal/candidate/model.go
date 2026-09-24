package candidate

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/fathdemr/nexus-interview/pkg/model"
	"gorm.io/gorm"
)

// Candidate represents a job applicant invited to take an AI-driven interview.
type Candidate struct {
	Id    string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	model.BaseRecordFields

	FullName        string     `gorm:"not null"                    json:"full_name"`
	Email           string     `gorm:"uniqueIndex;not null"        json:"email"`
	InviteToken     *string    `gorm:"index"                       json:"invite_token,omitempty"`
	InviteExpiresAt *time.Time `                                   json:"invite_expires_at,omitempty"`

	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (c *Candidate) BeforeCreate(_ *gorm.DB) error {
	if c.Id == "" {
		c.Id = uuid.Must(uuid.NewV4()).String()
	}
	return nil
}
