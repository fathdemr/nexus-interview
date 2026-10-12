package database

import (
	"fmt"

	"github.com/fathdemr/nexus-interview/internal/candidate"
	jobposting "github.com/fathdemr/nexus-interview/internal/job_posting"
	"github.com/fathdemr/nexus-interview/internal/question"
	"github.com/fathdemr/nexus-interview/pkg/model"
	"gorm.io/gorm"
)

// Migrate runs GORM AutoMigrate for all registered domain models.
// Register new models here as they are added to internal/.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&candidate.Candidate{},
		&question.Question{},
		&jobposting.JobPosting{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
