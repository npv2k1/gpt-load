package services

import (
	"context"
	"fmt"
	"gpt-load/internal/channel"
	"gpt-load/internal/models"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ModelService handles model-related operations
type ModelService struct {
	db             *gorm.DB
	channelFactory *channel.Factory
}

// NewModelService creates a new ModelService instance
func NewModelService(db *gorm.DB, channelFactory *channel.Factory) *ModelService {
	return &ModelService{
		db:             db,
		channelFactory: channelFactory,
	}
}

// FetchAndStoreModels fetches models from the provider and stores them in the database
func (s *ModelService) FetchAndStoreModels(ctx context.Context, group *models.Group, apiKey *models.APIKey) error {
	// Get the channel for this group
	ch, err := s.channelFactory.GetChannel(group)
	if err != nil {
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Fetch models from the provider
	capabilities, err := ch.FetchModels(ctx, apiKey, group)
	if err != nil {
		return fmt.Errorf("failed to fetch models: %w", err)
	}

	if len(capabilities) == 0 {
		return fmt.Errorf("no models returned from provider")
	}

	// Store or update models in database
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, capability := range capabilities {
			var existing models.ModelCapabilities
			result := tx.Where("group_id = ? AND model_id = ?", capability.GroupID, capability.ModelID).First(&existing)

			if result.Error == gorm.ErrRecordNotFound {
				// Create new record
				if err := tx.Create(&capability).Error; err != nil {
					logrus.WithFields(logrus.Fields{
						"group_id": capability.GroupID,
						"model_id": capability.ModelID,
						"error":    err,
					}).Error("Failed to create model capability")
					return err
				}
			} else if result.Error != nil {
				return result.Error
			} else {
				// Update existing record
				updates := map[string]interface{}{
					"model_name":         capability.ModelName,
					"supports_streaming": capability.SupportsStreaming,
					"supports_vision":    capability.SupportsVision,
					"supports_functions": capability.SupportsFunctions,
					"is_auto_fetched":    capability.IsAutoFetched,
					"last_fetched_at":    capability.LastFetchedAt,
				}

				if capability.MaxTokens != nil {
					updates["max_tokens"] = *capability.MaxTokens
				}
				if capability.MaxInputTokens != nil {
					updates["max_input_tokens"] = *capability.MaxInputTokens
				}
				if capability.MaxOutputTokens != nil {
					updates["max_output_tokens"] = *capability.MaxOutputTokens
				}

				if err := tx.Model(&existing).Updates(updates).Error; err != nil {
					logrus.WithFields(logrus.Fields{
						"group_id": capability.GroupID,
						"model_id": capability.ModelID,
						"error":    err,
					}).Error("Failed to update model capability")
					return err
				}
			}
		}
		return nil
	})
}

// GetModels retrieves all models for a group
func (s *ModelService) GetModels(groupID uint) ([]models.ModelCapabilities, error) {
	var capabilities []models.ModelCapabilities
	err := s.db.Where("group_id = ?", groupID).Order("model_name ASC").Find(&capabilities).Error
	return capabilities, err
}

// GetModelByID retrieves a specific model capability
func (s *ModelService) GetModelByID(id uint) (*models.ModelCapabilities, error) {
	var capability models.ModelCapabilities
	err := s.db.First(&capability, id).Error
	if err != nil {
		return nil, err
	}
	return &capability, nil
}

// UpdateModelCapability updates a model's custom capabilities
func (s *ModelService) UpdateModelCapability(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.ModelCapabilities{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteModel deletes a model capability
func (s *ModelService) DeleteModel(id uint) error {
	return s.db.Delete(&models.ModelCapabilities{}, id).Error
}

// DeleteGroupModels deletes all models for a group
func (s *ModelService) DeleteGroupModels(groupID uint) error {
	return s.db.Where("group_id = ?", groupID).Delete(&models.ModelCapabilities{}).Error
}

// GetAutoFetchedModels retrieves all auto-fetched models for a group
func (s *ModelService) GetAutoFetchedModels(groupID uint) ([]models.ModelCapabilities, error) {
	var capabilities []models.ModelCapabilities
	err := s.db.Where("group_id = ? AND is_auto_fetched = ?", groupID, true).
		Order("last_fetched_at DESC").
		Find(&capabilities).Error
	return capabilities, err
}

// RefreshStaleModels refreshes models that haven't been fetched recently
func (s *ModelService) RefreshStaleModels(ctx context.Context, group *models.Group, apiKey *models.APIKey, staleDuration time.Duration) error {
	// Check if we have any models that need refreshing
	var count int64
	staleTime := time.Now().Add(-staleDuration)
	
	err := s.db.Model(&models.ModelCapabilities{}).
		Where("group_id = ? AND is_auto_fetched = ? AND (last_fetched_at IS NULL OR last_fetched_at < ?)", 
			group.ID, true, staleTime).
		Count(&count).Error
	
	if err != nil {
		return err
	}

	// If we have stale models or no models at all, refresh
	if count > 0 {
		return s.FetchAndStoreModels(ctx, group, apiKey)
	}

	// Also refresh if we have no models at all
	var totalCount int64
	err = s.db.Model(&models.ModelCapabilities{}).Where("group_id = ?", group.ID).Count(&totalCount).Error
	if err != nil {
		return err
	}

	if totalCount == 0 {
		return s.FetchAndStoreModels(ctx, group, apiKey)
	}

	return nil
}
