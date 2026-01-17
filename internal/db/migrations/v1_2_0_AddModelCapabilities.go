package db

import (
	"gpt-load/internal/models"

	"gorm.io/gorm"
)

// V1_2_0_AddModelCapabilities adds the model_capabilities table for storing auto-fetched models
func V1_2_0_AddModelCapabilities(db *gorm.DB) error {
	return db.AutoMigrate(&models.ModelCapabilities{})
}
