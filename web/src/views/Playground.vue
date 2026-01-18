<script setup lang="ts">
import { getGroupList } from "@/api/dashboard";
import type { Group } from "@/types/models";
import { NButton, NCard, NInput, NSelect, NSpace, useMessage } from "naive-ui";
import { onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import http from "@/utils/http";

const { t } = useI18n();
const message = useMessage();

const groups = ref<Group[]>([]);
const selectedGroupId = ref<number | null>(null);
const userMessage = ref("");
const messages = ref<Array<{ role: string; content: string }>>([]);
const loading = ref(false);
const loadingGroups = ref(false);
const modelName = ref("gpt-4o-mini");
const temperature = ref("0.7");

const groupOptions = ref<Array<{ label: string; value: number }>>([]);

onMounted(async () => {
  await loadGroups();
});

async function loadGroups() {
  try {
    loadingGroups.value = true;
    const response = await getGroupList();
    groups.value = response.data;
    groupOptions.value = groups.value
      .filter(g => g.id !== undefined)
      .map(g => ({
        label: `${g.display_name || g.name} (${g.name})`,
        value: g.id as number,
      }));
    if (groups.value.length > 0 && groups.value[0].id !== undefined) {
      selectedGroupId.value = groups.value[0].id;
    }
  } catch (error) {
    console.error("Failed to load groups:", error);
    message.error(t("playground.failedToLoadGroups"));
  } finally {
    loadingGroups.value = false;
  }
}

async function sendMessage() {
  if (!userMessage.value.trim()) {
    message.warning(t("playground.pleaseEnterMessage"));
    return;
  }

  if (!selectedGroupId.value) {
    message.warning(t("playground.pleaseSelectGroup"));
    return;
  }

  const selectedGroup = groups.value.find(g => g.id === selectedGroupId.value);
  if (!selectedGroup) {
    message.error(t("playground.groupNotFound"));
    return;
  }

  // Validate temperature
  const tempValue = parseFloat(temperature.value) || 0.7;
  if (tempValue < 0 || tempValue > 2) {
    message.warning(t("playground.invalidTemperature"));
    return;
  }

  messages.value.push({
    role: "user",
    content: userMessage.value,
  });

  userMessage.value = "";
  loading.value = true;

  try {
    const response = await http.post(`/playground/chat`, {
      group_name: selectedGroup.name,
      model: modelName.value,
      messages: messages.value,
      temperature: tempValue,
    });

    if (response.data && response.data.content) {
      messages.value.push({
        role: "assistant",
        content: response.data.content,
      });
    } else {
      throw new Error("Invalid response format");
    }
  } catch (error: any) {
    console.error("Failed to send message:", error);
    messages.value.push({
      role: "error",
      content: error.response?.data?.error || error.message || t("playground.failedToSendMessage"),
    });
  } finally {
    loading.value = false;
  }
}

function clearMessages() {
  messages.value = [];
}
</script>

<template>
  <div class="playground-container">
    <n-space vertical size="large">
      <n-card :title="t('playground.title')" size="large">
        <template #header-extra>
          <n-button secondary @click="clearMessages">{{ t("playground.clearChat") }}</n-button>
        </template>

        <n-space vertical size="medium">
          <n-space>
            <n-select
              v-model:value="selectedGroupId"
              :options="groupOptions"
              :placeholder="t('playground.selectGroup')"
              :loading="loadingGroups"
              style="width: 250px"
            />
            <n-input
              v-model:value="modelName"
              :placeholder="t('playground.modelName')"
              style="width: 200px"
            />
            <n-input
              v-model:value="temperature"
              :placeholder="t('playground.temperature')"
              style="width: 120px"
            />
          </n-space>

          <div class="chat-container">
            <div v-if="messages.length === 0" class="empty-state">
              <div class="empty-icon">💬</div>
              <p>{{ t("playground.startConversation") }}</p>
            </div>

            <div v-else class="messages-list">
              <div
                v-for="(msg, index) in messages"
                :key="index"
                :class="['message', `message-${msg.role}`]"
              >
                <div class="message-role">
                  {{ msg.role === "user" ? "👤" : msg.role === "assistant" ? "🤖" : "❌" }}
                  {{ msg.role.toUpperCase() }}
                </div>
                <div class="message-content">{{ msg.content }}</div>
              </div>
            </div>
          </div>

          <n-space>
            <n-input
              v-model:value="userMessage"
              type="textarea"
              :placeholder="t('playground.enterMessage')"
              :rows="3"
              :disabled="loading"
              @keydown.ctrl.enter="sendMessage"
              style="flex: 1"
            />
            <n-button
              type="primary"
              :loading="loading"
              :disabled="!userMessage.trim() || !selectedGroupId"
              @click="sendMessage"
            >
              {{ loading ? t("playground.sending") : t("playground.send") }}
            </n-button>
          </n-space>

          <div class="hint">{{ t("playground.ctrlEnterHint") }}</div>
        </n-space>
      </n-card>
    </n-space>
  </div>
</template>

<style scoped>
.playground-container {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
}

.chat-container {
  min-height: 400px;
  max-height: 600px;
  overflow-y: auto;
  border: 1px solid var(--border-color, #e0e0e0);
  border-radius: 8px;
  padding: 16px;
  background: var(--bg-color, #fafafa);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 380px;
  color: var(--text-color-3, #999);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.messages-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message {
  padding: 12px;
  border-radius: 8px;
  animation: fadeIn 0.3s ease-in;
}

.message-user {
  background: var(--primary-color-hover, #e3f2fd);
  margin-left: 20%;
}

.message-assistant {
  background: var(--success-color-hover, #f1f8f4);
  margin-right: 20%;
}

.message-error {
  background: var(--error-color-hover, #ffebee);
  margin-right: 20%;
}

.message-role {
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 8px;
  opacity: 0.8;
}

.message-content {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
}

.hint {
  font-size: 12px;
  color: var(--text-color-3, #999);
  text-align: right;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
