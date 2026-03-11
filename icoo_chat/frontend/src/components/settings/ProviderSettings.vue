<template>
  <section class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-xl font-semibold mb-1">LLM 供应商</h2>
        <p class="text-text-secondary text-sm">管理 AI 模型提供商配置</p>
      </div>
      <button
        @click="openAddProvider"
        class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
      >
        <PlusIcon :size="16" />
        添加供应商
      </button>
    </div>

    <!-- AI Agent 默认模型设置 -->
    <div class="bg-bg-secondary rounded-xl border border-border p-4">
      <div class="flex items-start justify-between mb-3">
        <div>
          <h3 class="text-sm font-medium mb-1">AI Agent 默认模型</h3>
          <p class="text-xs text-text-secondary">
            设置 AI Agent 默认使用的模型，此设置优先于供应商 的默认模型
          </p>
        </div>
        <button
          @click="openAgentModelDialog"
          class="px-3 py-1.5 bg-accent hover:bg-accent-hover rounded-lg text-xs font-medium transition-colors flex items-center gap-1"
        >
          <EditIcon :size="14" />
          设置
        </button>
      </div>
      <div class="flex items-center gap-2 text-sm">
        <span class="text-text-secondary">当前模型：</span>
        <span v-if="agentDefaultModel" class="text-accent font-medium">
          {{ agentDefaultModel }}
        </span>
        <span v-else class="text-text-muted italic">
          未设置（将使用 Provider 的默认模型）
        </span>
      </div>
    </div>

    <div v-if="loading" class="text-text-secondary text-center py-8">
      加载中...
    </div>

    <div v-else-if="providers.length > 0" class="space-y-3">
      <div
        v-for="provider in providers"
        :key="provider.id"
        class="bg-bg-secondary rounded-xl border border-border p-4"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div
              :class="[
                'w-2 h-2 rounded-full',
                provider.enabled ? 'bg-green-500' : 'bg-text-muted',
              ]"
            ></div>
            <div>
              <div class="font-medium">{{ provider.name }}</div>
              <div class="text-xs text-text-secondary mt-0.5">
                {{ getProviderModel(provider) }}
              </div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <span
              :class="[
                'text-xs px-2 py-1 rounded',
                provider.enabled
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-bg-tertiary text-text-secondary',
              ]"
            >
              {{ provider.enabled ? "已启用" : "未启用" }}
            </span>
            <button
              @click="openEditProvider(provider)"
              class="p-1.5 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-accent transition-colors"
              title="编辑"
            >
              <EditIcon :size="16" />
            </button>
            <button
              @click="handleDeleteProvider(provider)"
              class="p-1.5 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-red-500 transition-colors"
              title="删除"
            >
              <TrashIcon :size="16" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <div
      v-else
      class="bg-bg-secondary rounded-xl border border-border p-8 text-center"
    >
      <div class="text-text-secondary text-sm mb-4">暂无供应商配置</div>
      <button
        @click="openAddProvider"
        class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
      >
        添加第一个供应商
      </button>
    </div>

    <!-- Provider 编辑弹窗 -->
    <ModalDialog
      v-model:visible="providerDialogVisible"
      :title="editingProvider ? '编辑供应商' : '添加供应商'"
      size="lg"
      :scrollable="true"
      :loading="savingProvider"
      :confirm-disabled="!providerForm.name"
      confirm-text="保存"
      loading-text="保存中..."
      @confirm="handleSaveProvider"
    >
      <div class="space-y-4">
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            供应商名称
          </label>
          <input
            v-model="providerForm.name"
            type="text"
            placeholder="例如: openai, anthropic, deepseek"
            :disabled="!!editingProvider"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors disabled:opacity-50"
          />
        </div>
        <div class="flex items-center gap-3">
          <input
            v-model="providerForm.enabled"
            type="checkbox"
            id="provider-enabled"
            class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
          />
          <label for="provider-enabled" class="text-sm"> 启用此供应商 </label>
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2"
            >供应商类型</label
          >
          <select
            v-model="providerForm.type"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
            placeholder="请选择供应商类型"
          >
            <option value="deepseek">DeepSeek</option>
            <option value="qwen">Qwen</option>
            <option value="qwen_coding_plan">Qwen Coding Plan</option>
            <option value="siliconflow">SiliconFlow</option>
            <option value="zhipu">Zhipu</option>
            <option value="openai">OpenAI</option>
            <option value="openrouter">OpenRouter</option>
            <option value="anthropic">Anthropic</option>
          </select>
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2">API Key</label>
          <input
            v-model="providerForm.api_key"
            type="password"
            placeholder="sk-..."
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2"
            >API Base URL</label
          >
          <input
            v-model="providerForm.api_base"
            type="text"
            placeholder="https://api.openai.com/v1"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
          <p class="text-xs text-text-secondary mt-1">
            可选，用于自定义 API 端点
          </p>
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            默认模型
          </label>
          <input
            v-model="providerForm.default_model"
            type="text"
            placeholder="例如：gpt-4, claude-sonnet-4-20250514"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
          <p class="text-xs text-text-secondary mt-1">
            可选，此供应商的默认模型。如果未设置，将使用第一个支持的模型。
          </p>
        </div>

        <!-- 模型列表 -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-sm text-text-secondary">支持的模型</label>
            <button
              @click="addModel"
              type="button"
              class="text-xs text-accent hover:text-accent-hover transition-colors flex items-center gap-1"
            >
              <PlusIcon :size="14" />
              添加模型
            </button>
          </div>

          <div v-if="providerForm.models.length > 0" class="space-y-2">
            <div
              v-for="(model, index) in providerForm.models"
              :key="index"
              class="flex items-center gap-2"
            >
              <input
                v-model="model.model"
                type="text"
                placeholder="模型名称 (如 gpt-4)"
                class="flex-1 px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
              />
              <input
                v-model="model.alias"
                type="text"
                placeholder="别名 (可选)"
                class="w-32 px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
              />
              <button
                @click="removeModel(index)"
                type="button"
                class="p-2 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-red-500 transition-colors"
              >
                <XIcon :size="16" />
              </button>
            </div>
          </div>

          <div
            v-else
            class="text-text-secondary text-sm py-3 text-center border border-dashed border-border rounded-lg"
          >
            暂无模型，点击上方按钮添加
          </div>
        </div>
      </div>
    </ModalDialog>

    <!-- AI Agent 默认模型弹窗 -->
    <ModalDialog
      v-model:visible="agentModelDialogVisible"
      title="设置 AI Agent 默认模型"
      size="md"
      :loading="savingAgentModel"
      confirm-text="保存"
      loading-text="保存中..."
      @confirm="handleSaveAgentModel"
    >
      <div class="space-y-4">
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            默认模型
          </label>
          <input
            v-model="agentModelForm.model"
            type="text"
            placeholder="例如：gpt-4, claude-sonnet-4-20250514"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
          <p class="text-xs text-text-secondary mt-1">
            设置后将优先使用此模型，优先级高于 Provider 的默认模型
          </p>
        </div>
        <div v-if="agentModelForm.providers.length > 0" class="space-y-2">
          <label class="block text-sm text-text-secondary mb-2">
            可选：从 Provider 支持的模型中选择
          </label>
          <div
            v-for="(provider, idx) in agentModelForm.providers"
            :key="idx"
            class="bg-bg-tertiary rounded-lg p-3"
          >
            <div class="text-xs font-medium text-text-secondary mb-2">
              {{ provider.name }}
            </div>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="(llm, llmIdx) in provider.llms"
                :key="llmIdx"
                @click="selectModel(provider.name, llm.model)"
                :class="[
                  'px-3 py-1.5 rounded-lg text-xs transition-colors',
                  agentModelForm.model === llm.model
                    ? 'bg-accent text-white'
                    : 'bg-bg-secondary text-text-secondary hover:bg-bg-hover',
                ]"
              >
                {{ `${provider.name}/${llm.model}` }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </ModalDialog>
  </section>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from "vue";
import {
  Plus as PlusIcon,
  Edit as EditIcon,
  Trash as TrashIcon,
  X as XIcon,
} from "lucide-vue-next";
import api from "@/services/api";
import ModalDialog from "@/components/ModalDialog.vue";

// Provider 数据
const providers = ref([]);
const loading = ref(true);

// AI Agent 默认模型数据
const agentDefaultModel = ref("");
const showAgentModelDialog = ref(false);
const savingAgentModel = ref(false);
const agentModelForm = reactive({
  model: "",
  providers: [],
});

// AI Agent 模型弹窗显示控制
const agentModelDialogVisible = computed({
  get: () => showAgentModelDialog.value,
  set: (val) => {
    if (!val) closeAgentModelDialog();
  },
});

// Provider 对话框
const showProviderDialog = ref(false);
const editingProvider = ref(null);
const savingProvider = ref(false);
const providerForm = reactive({
  name: "",
  enabled: true,
  api_key: "",
  api_base: "",
  default_model: "",
  models: [],
});

// Provider 弹窗显示控制
const providerDialogVisible = computed({
  get: () => showProviderDialog.value || !!editingProvider.value,
  set: (val) => {
    if (!val) closeProviderDialog();
  },
});

// 获取 Provider 模型名称
function getProviderModel(provider) {
  try {
    if (provider.default_model) {
      return provider.default_model + " (默认)";
    }
    if (provider.llms && provider.llms.length > 0) {
      const names = provider.llms.map((l) => l.alias || l.name);
      return names.slice(0, 3).join(", ") + (names.length > 3 ? "..." : "");
    }
    const config = JSON.parse(provider.config || "{}");
    return config.model || "-";
  } catch {
    return "-";
  }
}

// 加载 Provider 列表
async function loadProviders() {
  loading.value = true;
  try {
    const response = await api.getProviders();
    providers.value = response.data || [];
    await loadAgentDefaultModel();
  } catch (error) {
    console.error("获取 Provider 失败:", error);
    providers.value = [];
  }
  loading.value = false;
}

// 加载 AI Agent 默认模型
async function loadAgentDefaultModel() {
  try {
    const response = await api.getDefaultModel();
    if (response.data && response.data.model) {
      agentDefaultModel.value = response.data.model;
    }
  } catch (error) {
    console.error("获取 AI Agent 默认模型失败:", error);
  }
}

// 打开 AI Agent 默认模型对话框
function openAgentModelDialog() {
  agentModelForm.model = agentDefaultModel.value || "";
  agentModelForm.providers = providers.value.filter(
    (p) => p.enabled && p.llms && p.llms.length > 0,
  );
  showAgentModelDialog.value = true;
}

// 关闭 AI Agent 默认模型对话框
function closeAgentModelDialog() {
  showAgentModelDialog.value = false;
}

// 选择模型
function selectModel(provider, model) {
  agentModelForm.model = `${provider}/${model}`;
}

// 保存 AI Agent 默认模型
async function handleSaveAgentModel() {
  if (!agentModelForm.model) {
    alert("请输入模型名称");
    return;
  }

  savingAgentModel.value = true;
  try {
    await api.setDefaultModel({
      provider_id: null,
      model: agentModelForm.model,
    });
    agentDefaultModel.value = agentModelForm.model;
    closeAgentModelDialog();
    alert("AI Agent 默认模型设置成功");
  } catch (error) {
    console.error("设置 AI Agent 默认模型失败:", error);
    alert("设置失败：" + error.message);
  }
  savingAgentModel.value = false;
}

// 打开添加 Provider 对话框
function openAddProvider() {
  editingProvider.value = null;
  providerForm.name = "";
  providerForm.enabled = true;
  providerForm.api_key = "";
  providerForm.api_base = "";
  providerForm.default_model = "";
  providerForm.models = [];
  showProviderDialog.value = true;
}

// 打开编辑 Provider 对话框
function openEditProvider(provider) {
  editingProvider.value = provider;
  providerForm.name = provider.name;
  providerForm.enabled = provider.enabled;
  providerForm.api_key = provider.api_key || "";
  providerForm.api_base = provider.base_url || "";
  providerForm.default_model = provider.default_model || "";

  if (provider.llms && provider.llms.length > 0) {
    providerForm.models = provider.llms.map((l) => ({
      model: l.model || "",
      alias: l.model || "",
    }));
  } else {
    providerForm.models = [];
  }

  showProviderDialog.value = true;
}

// 关闭 Provider 对话框
function closeProviderDialog() {
  showProviderDialog.value = false;
  editingProvider.value = null;
}

// 添加模型
function addModel() {
  providerForm.models.push({ model: "", alias: "" });
}

// 删除模型
function removeModel(index) {
  providerForm.models.splice(index, 1);
}

// 保存 Provider
async function handleSaveProvider() {
  if (!providerForm.name) return;

  savingProvider.value = true;

  const llms = providerForm.models
    .filter((m) => m.model)
    .map((m) => ({
      model: m.model,
      alias: m.alias,
    }));

  const data = {
    name: providerForm.name,
    enabled: providerForm.enabled,
    api_key: providerForm.api_key,
    api_base: providerForm.api_base,
    default_model: providerForm.default_model,
    llms: llms,
  };

  try {
    if (editingProvider.value) {
      await api.updateProvider({
        id: editingProvider.value.id,
        ...data,
      });
    } else {
      await api.createProvider(data);
    }

    await loadProviders();
    closeProviderDialog();
  } catch (error) {
    console.error("保存 Provider 失败:", error);
    alert("保存 Provider 失败: " + error.message);
  }

  savingProvider.value = false;
}

// 删除 Provider
async function handleDeleteProvider(provider) {
  if (!confirm(`确定要删除 Provider "${provider.name}" 吗？`)) {
    return;
  }

  try {
    await api.deleteProvider(provider.id);
    await loadProviders();
  } catch (error) {
    console.error("删除 Provider 失败:", error);
    alert("删除 Provider 失败: " + error.message);
  }
}

onMounted(() => {
  loadProviders();
});
</script>
