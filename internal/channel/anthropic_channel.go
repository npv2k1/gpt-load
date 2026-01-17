package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	app_errors "gpt-load/internal/errors"
	"gpt-load/internal/models"
	"gpt-load/internal/utils"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	Register("anthropic", newAnthropicChannel)
}

type AnthropicChannel struct {
	*BaseChannel
}

func newAnthropicChannel(f *Factory, group *models.Group) (ChannelProxy, error) {
	base, err := f.newBaseChannel("anthropic", group)
	if err != nil {
		return nil, err
	}

	return &AnthropicChannel{
		BaseChannel: base,
	}, nil
}

// ModifyRequest sets the required headers for the Anthropic API.
func (ch *AnthropicChannel) ModifyRequest(req *http.Request, apiKey *models.APIKey, group *models.Group) {
	req.Header.Set("x-api-key", apiKey.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")
}

// IsStreamRequest checks if the request is for a streaming response using the pre-read body.
func (ch *AnthropicChannel) IsStreamRequest(c *gin.Context, bodyBytes []byte) bool {
	if strings.Contains(c.GetHeader("Accept"), "text/event-stream") {
		return true
	}

	if c.Query("stream") == "true" {
		return true
	}

	type streamPayload struct {
		Stream bool `json:"stream"`
	}
	var p streamPayload
	if err := json.Unmarshal(bodyBytes, &p); err == nil {
		return p.Stream
	}

	return false
}

func (ch *AnthropicChannel) ExtractModel(c *gin.Context, bodyBytes []byte) string {
	type modelPayload struct {
		Model string `json:"model"`
	}
	var p modelPayload
	if err := json.Unmarshal(bodyBytes, &p); err == nil {
		return p.Model
	}
	return ""
}

// ValidateKey checks if the given API key is valid by making a messages request.
func (ch *AnthropicChannel) ValidateKey(ctx context.Context, apiKey *models.APIKey, group *models.Group) (bool, error) {
	upstreamURL := ch.getUpstreamURL()
	if upstreamURL == nil {
		return false, fmt.Errorf("no upstream URL configured for channel %s", ch.Name)
	}

	// Parse validation endpoint to extract path and query parameters
	endpointURL, err := url.Parse(ch.ValidationEndpoint)
	if err != nil {
		return false, fmt.Errorf("failed to parse validation endpoint: %w", err)
	}

	// Build final URL with path and query parameters
	finalURL := *upstreamURL
	finalURL.Path = strings.TrimRight(finalURL.Path, "/") + endpointURL.Path
	finalURL.RawQuery = endpointURL.RawQuery
	reqURL := finalURL.String()

	// Use a minimal, low-cost payload for validation
	payload := gin.H{
		"model":      ch.TestModel,
		"max_tokens": 100,
		"messages": []gin.H{
			{"role": "user", "content": "hi"},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("failed to marshal validation payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return false, fmt.Errorf("failed to create validation request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	// Apply custom header rules if available
	if len(group.HeaderRuleList) > 0 {
		headerCtx := utils.NewHeaderVariableContext(group, apiKey)
		utils.ApplyHeaderRules(req, group.HeaderRuleList, headerCtx)
	}

	resp, err := ch.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send validation request: %w", err)
	}
	defer resp.Body.Close()

	// Any 2xx status code indicates the key is valid.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	// For non-200 responses, parse the body to provide a more specific error reason.
	errorBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("key is invalid (status %d), but failed to read error body: %w", resp.StatusCode, err)
	}

	// Use the new parser to extract a clean error message.
	parsedError := app_errors.ParseUpstreamError(errorBody)

	return false, fmt.Errorf("[status %d] %s", resp.StatusCode, parsedError)
}

// FetchModels fetches available models from the Anthropic provider.
func (ch *AnthropicChannel) FetchModels(ctx context.Context, apiKey *models.APIKey, group *models.Group) ([]models.ModelCapabilities, error) {
	upstreamURL := ch.getUpstreamURL()
	if upstreamURL == nil {
		return nil, fmt.Errorf("no upstream URL configured for channel %s", ch.Name)
	}

	// Build the models list endpoint URL
	finalURL := *upstreamURL
	finalURL.Path = strings.TrimRight(finalURL.Path, "/") + "/v1/models"
	reqURL := finalURL.String()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create models request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey.KeyValue)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	// Apply custom header rules if available
	if len(group.HeaderRuleList) > 0 {
		headerCtx := utils.NewHeaderVariableContext(group, apiKey)
		utils.ApplyHeaderRules(req, group.HeaderRuleList, headerCtx)
	}

	resp, err := ch.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send models request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(resp.Body)
		parsedError := app_errors.ParseUpstreamError(errorBody)
		return nil, fmt.Errorf("failed to fetch models [status %d]: %s", resp.StatusCode, parsedError)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read models response: %w", err)
	}

	var response struct {
		Data []struct {
			ID          string `json:"id"`
			Type        string `json:"type"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	capabilities := make([]models.ModelCapabilities, 0, len(response.Data))
	now := time.Now()

	for _, model := range response.Data {
		capability := models.ModelCapabilities{
			GroupID:           group.ID,
			ModelID:           model.ID,
			ModelName:         model.ID,
			SupportsStreaming: true, // Anthropic models support streaming
			IsAutoFetched:     true,
			LastFetchedAt:     &now,
		}

		// Claude models support vision in opus and sonnet variants
		if strings.Contains(model.ID, "claude-3") || strings.Contains(model.ID, "claude-sonnet") || strings.Contains(model.ID, "claude-opus") {
			capability.SupportsVision = true
		}

		capabilities = append(capabilities, capability)
	}

	return capabilities, nil
}
