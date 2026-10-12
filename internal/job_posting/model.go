package job_posting

import (
	"github.com/fathdemr/nexus-interview/pkg/model"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

// JobPostingStatus controls whether an ilan is visible/active.
type JobPostingStatus string

const (
	// StatusActive — ilan yayında, adaylar başvurabilir.
	StatusActive JobPostingStatus = "active"
	// StatusPassive — ilan pasif, adaylar başvuramaz (soft disable).
	StatusPassive JobPostingStatus = "passive"
)

// JobPosting represents a job opening created by an admin.
// Each posting groups a set of interview questions and can be
// shared with candidates via its shareable link.
type JobPosting struct {
	Id string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	Title       string           `gorm:"type:varchar(255);not null"   json:"title"`
	Description string           `gorm:"type:text"                    json:"description"`
	Department  string           `gorm:"type:varchar(100)"            json:"department"`
	Location    string           `gorm:"type:varchar(100)"            json:"location"`
	Status      JobPostingStatus `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`

	// ShareToken is a short opaque token embedded in the public share URL.
	// Generated once at creation; never changes. Format: /apply/<ShareToken>
	ShareToken string `gorm:"type:varchar(64);uniqueIndex;not null" json:"share_token"`

	model.BaseRecordFields
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (j *JobPosting) BeforeCreate(_ *gorm.DB) error {
	if j.Id == "" {
		j.Id = uuid.Must(uuid.NewV4()).String()
	}
	// Share token: first 8 chars of a new UUID (URL-friendly, collision risk negligible at this scale)
	if j.ShareToken == "" {
		j.ShareToken = uuid.Must(uuid.NewV4()).String()[:8]
	}
	return nil
}
