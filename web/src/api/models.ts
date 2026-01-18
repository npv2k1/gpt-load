import type { ModelCapability } from "@/types/models";
import http from "@/utils/http";

export const modelsApi = {
  // Fetch models from provider
  async fetchModels(groupId: number): Promise<{ models: ModelCapability[]; count: number }> {
    const res = await http.post("/models/fetch", { group_id: groupId });
    return res.data;
  },

  // List models for a group
  async getModels(groupId: number): Promise<{ models: ModelCapability[]; count: number }> {
    const res = await http.get(`/models/group/${groupId}`);
    return res.data;
  },

  // Get specific model
  async getModel(modelId: number): Promise<ModelCapability> {
    const res = await http.get(`/models/${modelId}`);
    return res.data;
  },

  // Update model capabilities
  async updateModel(
    modelId: number,
    data: {
      supports_streaming?: boolean;
      supports_vision?: boolean;
      supports_functions?: boolean;
      max_tokens?: number;
      max_input_tokens?: number;
      max_output_tokens?: number;
      custom_capabilities?: Record<string, unknown>;
    }
  ): Promise<ModelCapability> {
    const res = await http.put(`/models/${modelId}`, data);
    return res.data;
  },

  // Delete model
  async deleteModel(modelId: number): Promise<void> {
    await http.delete(`/models/${modelId}`);
  },

  // Refresh stale models
  async refreshModels(
    groupId: number,
    staleHours?: number
  ): Promise<{ models: ModelCapability[]; count: number }> {
    const params = staleHours ? `?stale_hours=${staleHours}` : "";
    const res = await http.post(`/models/group/${groupId}/refresh${params}`);
    return res.data;
  },
};
