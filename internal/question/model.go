package question

import (
	"github.com/fathdemr/nexus-interview/pkg/model"
	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

// Question represents a single interview question linked to a job posting.
type Question struct {
	Id string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	JobPostingId string `gorm:"not null;index;type:varchar(36)" json:"job_posting_id"`
	Text         string `gorm:"not null;type:text"              json:"text"`
	OrderIndex   int    `gorm:"not null;default:0"              json:"order_index"`

	model.BaseRecordFields
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (q *Question) BeforeCreate(_ *gorm.DB) error {
	if q.Id == "" {
		q.Id = uuid.Must(uuid.NewV4()).String()
	}
	return nil
}
