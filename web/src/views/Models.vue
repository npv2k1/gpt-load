<script setup lang="ts">
import { modelsApi } from "@/api/models";
import { keysApi } from "@/api/keys";
import GroupList from "@/components/keys/GroupList.vue";
import type { Group, ModelCapability } from "@/types/models";
import { onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  NButton,
  NCard,
  NDataTable,
  NSpace,
  NSpin,
  NTag,
  NModal,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSwitch,
  useDialog,
  useMessage,
  type DataTableColumns,
} from "naive-ui";
import { h } from "vue";
import { RefreshOutline, AddOutline, PencilOutline, TrashOutline } from "@vicons/ionicons5";

const { t } = useI18n();
const message = useMessage();
const dialog = useDialog();
const router = useRouter();
const route = useRoute();

const groups = ref<Group[]>([]);
const loading = ref(false);
const selectedGroup = ref<Group | null>(null);
const models = ref<ModelCapability[]>([]);
const modelsLoading = ref(false);
const showEditModal = ref(false);
const editingModel = ref<ModelCapability | null>(null);

// Form data for editing
const formData = ref({
  supports_streaming: false,
  supports_vision: false,
  supports_functions: false,
  max_tokens: undefined as number | undefined,
  max_input_tokens: undefined as number | undefined,
  max_output_tokens: undefined as number | undefined,
});

onMounted(async () => {
  await loadGroups();
});

async function loadGroups() {
  try {
    loading.value = true;
    groups.value = await keysApi.getGroups();
    
    // Select default group
    if (groups.value.length > 0 && !selectedGroup.value) {
      const groupId = route.query.groupId;
      const found = groups.value.find(g => String(g.id) === String(groupId));
      if (found) {
        handleGroupSelect(found);
      } else {
        handleGroupSelect(groups.value[0]);
      }
    }
  } catch (error) {
    console.error("Failed to load groups:", error);
    message.error(t("keys.load_groups_failed"));
  } finally {
    loading.value = false;
  }
}

async function loadModels() {
  if (!selectedGroup.value?.id) {
    models.value = [];
    return;
  }

  try {
    modelsLoading.value = true;
    const result = await modelsApi.getModels(selectedGroup.value.id);
    models.value = result.models || [];
  } catch (error) {
    console.error("Failed to load models:", error);
    message.error(t("models.load_failed"));
    models.value = [];
  } finally {
    modelsLoading.value = false;
  }
}

watch(selectedGroup, async (newGroup) => {
  if (newGroup?.id) {
    await loadModels();
  } else {
    models.value = [];
  }
});

function handleGroupSelect(group: Group | null) {
  selectedGroup.value = group || null;
  if (String(group?.id) !== String(route.query.groupId)) {
    router.push({ name: "models", query: { groupId: group?.id || "" } });
  }
}

async function handleFetchModels() {
  if (!selectedGroup.value?.id) return;

  try {
    modelsLoading.value = true;
    const result = await modelsApi.fetchModels(selectedGroup.value.id);
    message.success(t("models.fetch_success", { count: result.count }));
    await loadModels();
  } catch (error: any) {
    console.error("Failed to fetch models:", error);
    message.error(error.response?.data?.message || t("models.fetch_failed"));
  } finally {
    modelsLoading.value = false;
  }
}

async function handleRefreshModels() {
  if (!selectedGroup.value?.id) return;

  try {
    modelsLoading.value = true;
    const result = await modelsApi.refreshModels(selectedGroup.value.id, 24);
    message.success(t("models.refresh_success", { count: result.count }));
    await loadModels();
  } catch (error: any) {
    console.error("Failed to refresh models:", error);
    message.error(error.response?.data?.message || t("models.refresh_failed"));
  } finally {
    modelsLoading.value = false;
  }
}

function handleEditModel(model: ModelCapability) {
  editingModel.value = model;
  formData.value = {
    supports_streaming: model.supports_streaming,
    supports_vision: model.supports_vision,
    supports_functions: model.supports_functions,
    max_tokens: model.max_tokens || undefined,
    max_input_tokens: model.max_input_tokens || undefined,
    max_output_tokens: model.max_output_tokens || undefined,
  };
  showEditModal.value = true;
}

async function handleSaveModel() {
  if (!editingModel.value) return;

  try {
    await modelsApi.updateModel(editingModel.value.id, formData.value);
    message.success(t("models.update_success"));
    showEditModal.value = false;
    await loadModels();
  } catch (error: any) {
    console.error("Failed to update model:", error);
    message.error(error.response?.data?.message || t("models.update_failed"));
  }
}

function handleDeleteModel(model: ModelCapability) {
  dialog.warning({
    title: t("models.delete_confirm_title"),
    content: t("models.delete_confirm_content", { modelName: model.model_name }),
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      try {
        await modelsApi.deleteModel(model.id);
        message.success(t("models.delete_success"));
        await loadModels();
      } catch (error: any) {
        console.error("Failed to delete model:", error);
        message.error(error.response?.data?.message || t("models.delete_failed"));
      }
    },
  });
}

const columns: DataTableColumns<ModelCapability> = [
  {
    title: t("models.model_id"),
    key: "model_id",
    width: 200,
    ellipsis: {
      tooltip: true,
    },
  },
  {
    title: t("models.model_name"),
    key: "model_name",
    width: 200,
    ellipsis: {
      tooltip: true,
    },
  },
  {
    title: t("models.capabilities"),
    key: "capabilities",
    width: 280,
    render: (row) => {
      return h(
        NSpace,
        { size: [4, 4], wrap: true },
        {
          default: () => [
            row.supports_streaming &&
              h(NTag, { type: "info", size: "small" }, { default: () => t("models.streaming") }),
            row.supports_vision &&
              h(NTag, { type: "success", size: "small" }, { default: () => t("models.vision") }),
            row.supports_functions &&
              h(NTag, { type: "warning", size: "small" }, { default: () => t("models.functions") }),
          ].filter(Boolean),
        }
      );
    },
  },
  {
    title: t("models.token_limits"),
    key: "tokens",
    width: 200,
    render: (row) => {
      const parts = [];
      if (row.max_tokens) parts.push(`Max: ${row.max_tokens.toLocaleString()}`);
      if (row.max_input_tokens) parts.push(`In: ${row.max_input_tokens.toLocaleString()}`);
      if (row.max_output_tokens) parts.push(`Out: ${row.max_output_tokens.toLocaleString()}`);
      return parts.length > 0 ? parts.join(" | ") : "-";
    },
  },
  {
    title: t("models.source"),
    key: "is_auto_fetched",
    width: 120,
    render: (row) => {
      return h(
        NTag,
        { type: row.is_auto_fetched ? "info" : "default", size: "small" },
        { default: () => (row.is_auto_fetched ? t("models.auto_fetched") : t("models.manual")) }
      );
    },
  },
  {
    title: t("models.last_fetched"),
    key: "last_fetched_at",
    width: 180,
    render: (row) => {
      if (!row.last_fetched_at) return "-";
      return new Date(row.last_fetched_at).toLocaleString();
    },
  },
  {
    title: t("common.actions"),
    key: "actions",
    width: 150,
    fixed: "right",
    render: (row) => {
      return h(
        NSpace,
        { size: [4, 4] },
        {
          default: () => [
            h(
              NButton,
              {
                size: "small",
                tertiary: true,
                onClick: () => handleEditModel(row),
              },
              {
                icon: () => h(PencilOutline),
              }
            ),
            h(
              NButton,
              {
                size: "small",
                tertiary: true,
                type: "error",
                onClick: () => handleDeleteModel(row),
              },
              {
                icon: () => h(TrashOutline),
              }
            ),
          ],
        }
      );
    },
  },
];
</script>

<template>
  <div class="models-container">
    <n-space vertical :size="16">
      <!-- Groups Sidebar -->
      <n-card :title="t('models.title')" size="small">
        <n-space vertical :size="12">
          <GroupList
            :groups="groups"
            :selected-group="selectedGroup"
            :loading="loading"
            @select="handleGroupSelect"
            @refresh="loadGroups"
          />
        </n-space>
      </n-card>

      <!-- Models Content -->
      <n-card v-if="selectedGroup" size="small">
        <template #header>
          <n-space justify="space-between" align="center">
            <span>{{ t("models.models_list") }} - {{ selectedGroup.display_name || selectedGroup.name }}</span>
            <n-space>
              <n-button
                type="primary"
                size="small"
                @click="handleFetchModels"
                :loading="modelsLoading"
              >
                <template #icon>
                  <AddOutline />
                </template>
                {{ t("models.fetch_models") }}
              </n-button>
              <n-button
                size="small"
                @click="handleRefreshModels"
                :loading="modelsLoading"
              >
                <template #icon>
                  <RefreshOutline />
                </template>
                {{ t("models.refresh_models") }}
              </n-button>
            </n-space>
          </n-space>
        </template>

        <n-spin :show="modelsLoading">
          <n-data-table
            :columns="columns"
            :data="models"
            :scroll-x="1400"
            :pagination="{ pageSize: 20 }"
            size="small"
          />
        </n-spin>
      </n-card>

      <!-- Empty State -->
      <n-card v-else size="small">
        <n-space vertical align="center" :size="12" style="padding: 40px">
          <span style="font-size: 48px">🤖</span>
          <span>{{ t("models.select_group_hint") }}</span>
        </n-space>
      </n-card>
    </n-space>

    <!-- Edit Model Modal -->
    <n-modal
      v-model:show="showEditModal"
      preset="card"
      :title="t('models.edit_model')"
      style="width: 600px"
    >
      <n-form>
        <n-form-item :label="t('models.model_id')">
          <n-input :value="editingModel?.model_id" disabled />
        </n-form-item>
        <n-form-item :label="t('models.model_name')">
          <n-input :value="editingModel?.model_name" disabled />
        </n-form-item>
        <n-form-item :label="t('models.supports_streaming')">
          <n-switch v-model:value="formData.supports_streaming" />
        </n-form-item>
        <n-form-item :label="t('models.supports_vision')">
          <n-switch v-model:value="formData.supports_vision" />
        </n-form-item>
        <n-form-item :label="t('models.supports_functions')">
          <n-switch v-model:value="formData.supports_functions" />
        </n-form-item>
        <n-form-item :label="t('models.max_tokens')">
          <n-input-number
            v-model:value="formData.max_tokens"
            :min="0"
            :placeholder="t('common.optional')"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item :label="t('models.max_input_tokens')">
          <n-input-number
            v-model:value="formData.max_input_tokens"
            :min="0"
            :placeholder="t('common.optional')"
            style="width: 100%"
          />
        </n-form-item>
        <n-form-item :label="t('models.max_output_tokens')">
          <n-input-number
            v-model:value="formData.max_output_tokens"
            :min="0"
            :placeholder="t('common.optional')"
            style="width: 100%"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">{{ t("common.cancel") }}</n-button>
          <n-button type="primary" @click="handleSaveModel">{{ t("common.save") }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.models-container {
  padding: 16px;
  max-width: 1600px;
  margin: 0 auto;
}
</style>
