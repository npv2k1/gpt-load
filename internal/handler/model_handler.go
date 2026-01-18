// Package handler provides HTTP handlers for the application
package handler

import (
	"strconv"
	"time"

	app_errors "gpt-load/internal/errors"
	"gpt-load/internal/i18n"
	"gpt-load/internal/models"
	"gpt-load/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// FetchModelsRequest defines the request payload for fetching models
type FetchModelsRequest struct {
	GroupID uint `json:"group_id" binding:"required"`
}

// UpdateModelRequest defines the request payload for updating model capabilities
type UpdateModelRequest struct {
	SupportsStreaming  *bool                  `json:"supports_streaming"`
	SupportsVision     *bool                  `json:"supports_vision"`
	SupportsFunctions  *bool                  `json:"supports_functions"`
	MaxTokens          *int                   `json:"max_tokens"`
	MaxInputTokens     *int                   `json:"max_input_tokens"`
	MaxOutputTokens    *int                   `json:"max_output_tokens"`
	CustomCapabilities map[string]interface{} `json:"custom_capabilities"`
}

// FetchModels handles fetching models from the provider
func (s *Server) FetchModels(c *gin.Context) {
	var req FetchModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	// Get the group
	var group models.Group
	if err := s.DB.First(&group, req.GroupID).Error; err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ParseDBError(err), "group.group_not_found")
		return
	}

	// Get the effective config for the group
	effectiveConfig := s.SettingsManager.GetEffectiveConfig(group.Config)
	group.EffectiveConfig = effectiveConfig

	// Get an active API key for this group
	var apiKey models.APIKey
	err := s.DB.Where("group_id = ? AND status = ?", req.GroupID, models.KeyStatusActive).
		First(&apiKey).Error
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrNoActiveKeys, "key.no_active_keys")
		return
	}

	// Fetch and store models
	if err := s.ModelService.FetchAndStoreModels(c.Request.Context(), &group, &apiKey); err != nil {
		logrus.WithError(err).Error("Failed to fetch models")
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to fetch models: "+err.Error()))
		return
	}

	// Return the fetched models
	capabilities, err := s.ModelService.GetModels(req.GroupID)
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, err.Error()))
		return
	}

	response.Success(c, gin.H{
		"models": capabilities,
		"count":  len(capabilities),
	})
}

// ListModels handles listing all models for a group
func (s *Server) ListModels(c *gin.Context) {
	groupIDStr := c.Param("groupId")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrBadRequest, "validation.invalid_group_id")
		return
	}

	capabilities, err := s.ModelService.GetModels(uint(groupID))
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, err.Error()))
		return
	}

	response.Success(c, gin.H{
		"models": capabilities,
		"count":  len(capabilities),
	})
}

// GetModel handles retrieving a specific model
func (s *Server) GetModel(c *gin.Context) {
	modelIDStr := c.Param("modelId")
	modelID, err := strconv.ParseUint(modelIDStr, 10, 64)
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrBadRequest, "validation.invalid_model_id")
		return
	}

	capability, err := s.ModelService.GetModelByID(uint(modelID))
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrResourceNotFound, "model.model_not_found")
		return
	}

	response.Success(c, capability)
}

// UpdateModel handles updating a model's custom capabilities
func (s *Server) UpdateModel(c *gin.Context) {
	modelIDStr := c.Param("modelId")
	modelID, err := strconv.ParseUint(modelIDStr, 10, 64)
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrBadRequest, "validation.invalid_model_id")
		return
	}

	var req UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInvalidJSON, err.Error()))
		return
	}

	updates := make(map[string]interface{})
	if req.SupportsStreaming != nil {
		updates["supports_streaming"] = *req.SupportsStreaming
	}
	if req.SupportsVision != nil {
		updates["supports_vision"] = *req.SupportsVision
	}
	if req.SupportsFunctions != nil {
		updates["supports_functions"] = *req.SupportsFunctions
	}
	if req.MaxTokens != nil {
		updates["max_tokens"] = *req.MaxTokens
	}
	if req.MaxInputTokens != nil {
		updates["max_input_tokens"] = *req.MaxInputTokens
	}
	if req.MaxOutputTokens != nil {
		updates["max_output_tokens"] = *req.MaxOutputTokens
	}
	if req.CustomCapabilities != nil {
		updates["custom_capabilities"] = req.CustomCapabilities
	}

	if err := s.ModelService.UpdateModelCapability(uint(modelID), updates); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, err.Error()))
		return
	}

	// Get the updated model
	capability, err := s.ModelService.GetModelByID(uint(modelID))
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrResourceNotFound, "model.model_not_found")
		return
	}

	response.Success(c, capability)
}

// DeleteModel handles deleting a model
func (s *Server) DeleteModel(c *gin.Context) {
	modelIDStr := c.Param("modelId")
	modelID, err := strconv.ParseUint(modelIDStr, 10, 64)
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrBadRequest, "validation.invalid_model_id")
		return
	}

	if err := s.ModelService.DeleteModel(uint(modelID)); err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, err.Error()))
		return
	}

	response.Success(c, gin.H{
		"message": i18n.Message(c, "model.deleted_successfully"),
	})
}

// RefreshModels handles refreshing stale models for a group
func (s *Server) RefreshModels(c *gin.Context) {
	groupIDStr := c.Param("groupId")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrBadRequest, "validation.invalid_group_id")
		return
	}

	// Get the group
	var group models.Group
	if err := s.DB.First(&group, groupID).Error; err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ParseDBError(err), "group.group_not_found")
		return
	}

	// Get the effective config for the group
	effectiveConfig := s.SettingsManager.GetEffectiveConfig(group.Config)
	group.EffectiveConfig = effectiveConfig

	// Get an active API key for this group
	var apiKey models.APIKey
	err = s.DB.Where("group_id = ? AND status = ?", groupID, models.KeyStatusActive).
		First(&apiKey).Error
	if err != nil {
		response.ErrorI18nFromAPIError(c, app_errors.ErrNoActiveKeys, "key.no_active_keys")
		return
	}

	// Get staleness duration from query parameter, default to 24 hours
	staleDurationHours := 24
	if hoursStr := c.Query("stale_hours"); hoursStr != "" {
		if hours, err := strconv.Atoi(hoursStr); err == nil && hours > 0 {
			staleDurationHours = hours
		}
	}
	staleDuration := time.Duration(staleDurationHours) * time.Hour

	// Refresh models
	if err := s.ModelService.RefreshStaleModels(c.Request.Context(), &group, &apiKey, staleDuration); err != nil {
		logrus.WithError(err).Error("Failed to refresh models")
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, "Failed to refresh models: "+err.Error()))
		return
	}

	// Return the refreshed models
	capabilities, err := s.ModelService.GetModels(uint(groupID))
	if err != nil {
		response.Error(c, app_errors.NewAPIError(app_errors.ErrInternalServer, err.Error()))
		return
	}

	response.Success(c, gin.H{
		"models": capabilities,
		"count":  len(capabilities),
	})
}
