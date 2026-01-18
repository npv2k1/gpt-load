# Auto-Fetch Model Feature

## Overview

The auto-fetch model feature allows GPT-Load to automatically fetch and store available models from AI providers (OpenAI, Gemini, and Anthropic).

## API Endpoints

### 1. Fetch Models from Provider
POST /api/models/fetch

### 2. List Models for a Group
GET /api/models/group/:groupId

### 3. Get Specific Model
GET /api/models/:modelId

### 4. Update Model Capabilities
PUT /api/models/:modelId

### 5. Delete Model
DELETE /api/models/:modelId

### 6. Refresh Stale Models
POST /api/models/group/:groupId/refresh?stale_hours=24

## Usage Example

```bash
curl -X POST http://localhost:3001/api/models/fetch \
  -H "Authorization: Bearer your-auth-key" \
  -H "Content-Type: application/json" \
  -d '{"group_id": 1}'
```

See full documentation for detailed API specs and examples.
