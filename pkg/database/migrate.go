package database

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate runs GORM AutoMigrate for all registered domain models.
// Register new models here as they are added to internal/.
func Migrate(db *gorm.DB) error {
	// models will be added here as internal modules are built
	// Example: db.AutoMigrate(&candidate.Candidate{}, &interview.Interview{})
	if err := db.AutoMigrate(); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
