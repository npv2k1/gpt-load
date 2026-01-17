package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gpt-load/internal/models"
	"gpt-load/internal/response"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PlaygroundChatRequest represents the chat request from playground
type PlaygroundChatRequest struct {
	GroupName   string                   `json:"group_name" binding:"required"`
	Model       string                   `json:"model" binding:"required"`
	Messages    []map[string]interface{} `json:"messages" binding:"required"`
	Temperature float64                  `json:"temperature"`
}

// PlaygroundChatResponse represents the response to playground
type PlaygroundChatResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
}

// PlaygroundChat handles chat requests from the playground
func (s *Server) PlaygroundChat(c *gin.Context) {
	var req PlaygroundChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Find the group
	var group models.Group
	if err := s.DB.Where("name = ?", req.GroupName).First(&group).Error; err != nil {
		response.Error(c, http.StatusNotFound, fmt.Sprintf("Group '%s' not found", req.GroupName))
		return
	}

	// Get the first upstream from the group
	if len(group.Upstreams) == 0 {
		response.Error(c, http.StatusBadRequest, "Group has no upstreams configured")
		return
	}

	upstream := group.Upstreams[0]

	// Get an API key from the group
	var apiKey models.APIKey
	if err := s.DB.Where("group_id = ? AND status = ?", group.ID, models.KeyStatusActive).
		Order("RANDOM()").First(&apiKey).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "No active API keys available in this group")
		return
	}

	// Decrypt the API key
	decryptedKey, err := s.EncryptionSvc.Decrypt(apiKey.KeyValue)
	if err != nil {
		logrus.WithError(err).Error("Failed to decrypt API key")
		response.Error(c, http.StatusInternalServerError, "Failed to decrypt API key")
		return
	}

	// Build the request based on channel type
	var upstreamResp string
	var apiErr error

	switch group.ChannelType {
	case "openai":
		upstreamResp, apiErr = s.callOpenAI(upstream.URL, decryptedKey, req)
	case "gemini":
		upstreamResp, apiErr = s.callGemini(upstream.URL, decryptedKey, req)
	case "anthropic":
		upstreamResp, apiErr = s.callAnthropic(upstream.URL, decryptedKey, req)
	default:
		// Default to OpenAI format
		upstreamResp, apiErr = s.callOpenAI(upstream.URL, decryptedKey, req)
	}

	if apiErr != nil {
		response.Error(c, http.StatusInternalServerError, fmt.Sprintf("API call failed: %s", apiErr.Error()))
		return
	}

	response.Success(c, PlaygroundChatResponse{
		Content: upstreamResp,
		Model:   req.Model,
	})
}

func (s *Server) callOpenAI(baseURL, apiKey string, req PlaygroundChatRequest) (string, error) {
	// Build OpenAI chat completion request
	reqBody := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Extract the message content
	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (s *Server) callGemini(baseURL, apiKey string, req PlaygroundChatRequest) (string, error) {
	// Convert OpenAI messages format to Gemini format
	var contents []map[string]interface{}
	for _, msg := range req.Messages {
		role := "user"
		if msgRole, ok := msg["role"].(string); ok && msgRole == "assistant" {
			role = "model"
		}
		if content, ok := msg["content"].(string); ok {
			contents = append(contents, map[string]interface{}{
				"role": role,
				"parts": []map[string]interface{}{
					{"text": content},
				},
			})
		}
	}

	reqBody := map[string]interface{}{
		"contents": contents,
		"generationConfig": map[string]interface{}{
			"temperature": req.Temperature,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, req.Model, apiKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Extract the message content from Gemini response
	if candidates, ok := result["candidates"].([]interface{}); ok && len(candidates) > 0 {
		if candidate, ok := candidates[0].(map[string]interface{}); ok {
			if content, ok := candidate["content"].(map[string]interface{}); ok {
				if parts, ok := content["parts"].([]interface{}); ok && len(parts) > 0 {
					if part, ok := parts[0].(map[string]interface{}); ok {
						if text, ok := part["text"].(string); ok {
							return text, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (s *Server) callAnthropic(baseURL, apiKey string, req PlaygroundChatRequest) (string, error) {
	// Convert OpenAI messages to Anthropic format
	var messages []map[string]interface{}
	for _, msg := range req.Messages {
		if content, ok := msg["content"].(string); ok {
			messages = append(messages, map[string]interface{}{
				"role":    msg["role"],
				"content": content,
			})
		}
	}

	reqBody := map[string]interface{}{
		"model":       req.Model,
		"messages":    messages,
		"max_tokens":  1024,
		"temperature": req.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := baseURL + "/v1/messages"
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	// Extract the message content from Anthropic response
	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if textBlock, ok := content[0].(map[string]interface{}); ok {
			if text, ok := textBlock["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}
