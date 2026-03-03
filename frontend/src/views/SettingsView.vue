<template>
    <div class="w-full min-h-screen bg-bg-primary text-text-primary flex">
        <!-- 左侧导航 -->
        <aside class="w-64 border-r border-border bg-bg-secondary flex-shrink-0">
            <div class="p-4 border-b border-border">
                <div class="flex items-center gap-2">
                    <button
                        @click="router.back()"
                        class="p-1.5 rounded-lg hover:bg-bg-tertiary transition-colors"
                    >
                        <ArrowLeftIcon :size="18" />
                    </button>
                    <h1 class="text-lg font-semibold">设置</h1>
                </div>
            </div>

            <nav class="p-2">
                <button
                    v-for="item in menuItems"
                    :key="item.key"
                    @click="activeSection = item.key"
                    :class="[
                        'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors',
                        activeSection === item.key
                            ? 'bg-accent/10 text-accent'
                            : 'text-text-secondary hover:bg-bg-tertiary hover:text-text-primary'
                    ]"
                >
                    <component :is="item.icon" :size="18" />
                    <span class="text-sm font-medium">{{ item.label }}</span>
                </button>
            </nav>
        </aside>

        <!-- 右侧内容 -->
        <main class="flex-1 overflow-y-auto">
            <div class="max-w-3xl mx-auto px-6 py-8">
                <!-- 连接设置 -->
                <section v-if="activeSection === 'connection'" class="space-y-6">
                    <div>
                        <h2 class="text-xl font-semibold mb-1">连接设置</h2>
                        <p class="text-text-secondary text-sm">配置 API 和 WebSocket 连接地址</p>
                    </div>

                    <div class="bg-bg-secondary rounded-xl border border-border p-6 space-y-4">
                        <div>
                            <label class="block text-sm text-text-secondary mb-2">
                                WebSocket 地址
                            </label>
                            <input
                                v-model="wsUrl"
                                type="text"
                                placeholder="ws://localhost:8080/ws"
                                class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                            />
                        </div>

                        <div>
                            <label class="block text-sm text-text-secondary mb-2">
                                API 基础地址
                            </label>
                            <input
                                v-model="apiBase"
                                type="text"
                                placeholder="http://localhost:8080"
                                class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                            />
                        </div>

                        <div>
                            <label class="block text-sm text-text-secondary mb-2">
                                用户 ID
                            </label>
                            <input
                                v-model="userId"
                                type="text"
                                placeholder="user-1"
                                class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                            />
                        </div>
                    </div>

                    <!-- 连接状态 -->
                    <div class="bg-bg-secondary rounded-xl border border-border p-6">
                        <h3 class="text-sm font-medium mb-4">连接状态</h3>
                        <div class="space-y-3">
                            <div class="flex items-center justify-between">
                                <span class="text-text-secondary text-sm">API 状态</span>
                                <span
                                    :class="[
                                        'text-sm',
                                        apiHealth === 'ok' ? 'text-green-500' : 'text-red-500'
                                    ]"
                                >
                                    {{ apiHealth === "ok" ? "已连接" : "未连接" }}
                                </span>
                            </div>
                            <div class="flex items-center justify-between">
                                <span class="text-text-secondary text-sm">WebSocket</span>
                                <span class="text-text-secondary text-sm">{{ wsStatus }}</span>
                            </div>
                        </div>
                    </div>
                </section>

                <!-- 工作区设置 -->
                <section v-if="activeSection === 'workspace'" class="space-y-6">
                    <div>
                        <h2 class="text-xl font-semibold mb-1">工作区设置</h2>
                        <p class="text-text-secondary text-sm">配置工作目录路径</p>
                    </div>

                    <div class="bg-bg-secondary rounded-xl border border-border p-6 space-y-4">
                        <div>
                            <label class="block text-sm text-text-secondary mb-2">
                                工作区路径
                            </label>
                            <div class="flex gap-2">
                                <input
                                    v-model="workspace"
                                    type="text"
                                    placeholder="./workspace"
                                    class="flex-1 px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                                />
                                <button
                                    @click="handleSetWorkspace"
                                    :disabled="savingWorkspace"
                                    class="px-4 py-2.5 bg-accent hover:bg-accent-hover disabled:opacity-50 rounded-lg text-sm font-medium transition-colors"
                                >
                                    {{ savingWorkspace ? "保存中..." : "保存" }}
                                </button>
                            </div>
                            <p class="text-xs text-text-secondary mt-2">
                                修改工作区后需要重启服务才能生效
                            </p>
                        </div>
                    </div>

                    <!-- 配置文件 -->
                    <div class="bg-bg-secondary rounded-xl border border-border p-6">
                        <div class="flex items-center justify-between mb-4">
                            <h3 class="text-sm font-medium">配置文件</h3>
                            <button
                                @click="loadConfigFile"
                                class="text-xs text-accent hover:text-accent-hover transition-colors"
                            >
                                刷新
                            </button>
                        </div>

                        <div class="mb-3">
                            <label class="block text-xs text-text-secondary mb-1">文件路径</label>
                            <div class="text-sm text-text-secondary">{{ configPath || '-' }}</div>
                        </div>

                        <div>
                            <label class="block text-xs text-text-secondary mb-2">配置内容 (TOML)</label>
                            <textarea
                                v-model="configContent"
                                rows="12"
                                class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm font-mono focus:outline-none focus:border-accent transition-colors resize-none"
                                placeholder="加载配置文件..."
                            ></textarea>
                        </div>

                        <div class="flex justify-end gap-2 mt-3">
                            <button
                                @click="loadConfigFile"
                                class="px-3 py-1.5 text-sm border border-border rounded-lg hover:bg-bg-tertiary transition-colors"
                            >
                                重置
                            </button>
                            <button
                                @click="handleOverwriteConfig"
                                :disabled="savingConfig"
                                class="px-3 py-1.5 text-sm bg-accent hover:bg-accent-hover disabled:opacity-50 rounded-lg transition-colors"
                            >
                                {{ savingConfig ? "保存中..." : "保存配置" }}
                            </button>
                        </div>
                    </div>
                </section>

                <!-- Provider 设置 -->
                <section v-if="activeSection === 'provider'" class="space-y-6">
                    <div class="flex items-center justify-between">
                        <div>
                            <h2 class="text-xl font-semibold mb-1">LLM Provider</h2>
                            <p class="text-text-secondary text-sm">管理 AI 模型提供商配置</p>
                        </div>
                        <button
                            @click="openAddProvider"
                            class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                        >
                            <PlusIcon :size="16" />
                            添加 Provider
                        </button>
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
                                            provider.enabled ? 'bg-green-500' : 'bg-text-muted'
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
                                                : 'bg-bg-tertiary text-text-secondary'
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

                    <div v-else class="bg-bg-secondary rounded-xl border border-border p-8 text-center">
                        <div class="text-text-secondary text-sm mb-4">
                            暂无 Provider 配置
                        </div>
                        <button
                            @click="openAddProvider"
                            class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
                        >
                            添加第一个 Provider
                        </button>
                    </div>
                </section>

                <!-- 技能管理 -->
                <section v-if="activeSection === 'skill'" class="space-y-6">
                    <div class="flex items-center justify-between">
                        <div>
                            <h2 class="text-xl font-semibold mb-1">技能管理</h2>
                            <p class="text-text-secondary text-sm">管理自定义技能，扩展 AI 助手能力</p>
                        </div>
                        <button
                            @click="router.push('/skills')"
                            class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
                        >
                            管理技能
                        </button>
                    </div>

                    <div class="bg-bg-secondary rounded-xl border border-border p-4">
                        <div class="flex items-center justify-between">
                            <div>
                                <div class="text-sm font-medium">已启用技能</div>
                                <div class="text-xs text-text-secondary mt-0.5">
                                    {{ skillStore.enabledSkills.length }} / {{ skillStore.skills.length }}
                                </div>
                            </div>
                            <ChevronRightIcon :size="18" class="text-text-secondary" />
                        </div>
                    </div>
                </section>

                <!-- 外观设置 -->
                <section v-if="activeSection === 'appearance'" class="space-y-6">
                    <div>
                        <h2 class="text-xl font-semibold mb-1">外观设置</h2>
                        <p class="text-text-secondary text-sm">自定义界面外观</p>
                    </div>

                    <div class="bg-bg-secondary rounded-xl border border-border p-6">
                        <div class="flex items-center justify-between">
                            <div>
                                <div class="font-medium">主题模式</div>
                                <div class="text-sm text-text-secondary mt-1">
                                    切换明暗主题
                                </div>
                            </div>
                            <div class="flex items-center gap-1 bg-bg-tertiary rounded-lg p-1">
                                <button
                                    @click="themeStore.setTheme('light')"
                                    :class="[
                                        'px-3 py-1.5 rounded-md text-sm transition-colors flex items-center gap-1.5',
                                        themeStore.theme === 'light'
                                            ? 'bg-accent text-white'
                                            : 'text-text-secondary hover:text-text-primary'
                                    ]"
                                >
                                    <SunIcon :size="14" />
                                    浅色
                                </button>
                                <button
                                    @click="themeStore.setTheme('dark')"
                                    :class="[
                                        'px-3 py-1.5 rounded-md text-sm transition-colors flex items-center gap-1.5',
                                        themeStore.theme === 'dark'
                                            ? 'bg-accent text-white'
                                            : 'text-text-secondary hover:text-text-primary'
                                    ]"
                                >
                                    <MoonIcon :size="14" />
                                    深色
                                </button>
                            </div>
                        </div>
                    </div>
                </section>

                <!-- 关于 -->
                <section v-if="activeSection === 'about'" class="space-y-6">
                    <div>
                        <h2 class="text-xl font-semibold mb-1">关于</h2>
                        <p class="text-text-secondary text-sm">icooclaw 版本信息</p>
                    </div>

                    <div class="bg-bg-secondary rounded-xl border border-border p-6">
                        <div class="text-center">
                            <div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-accent/20 flex items-center justify-center">
                                <SparklesIcon :size="32" class="text-accent" />
                            </div>
                            <h3 class="text-lg font-semibold">icooclaw</h3>
                            <p class="text-text-secondary text-sm mt-1">AI 助手平台</p>
                            <p class="text-text-muted text-xs mt-2">版本 1.0.0</p>
                        </div>
                    </div>
                </section>

                <!-- 底部保存按钮 -->
                <div v-if="hasChanges" class="fixed bottom-0 left-64 right-0 bg-bg-secondary border-t border-border p-4">
                    <div class="max-w-3xl mx-auto flex justify-end gap-3">
                        <button
                            @click="handleReset"
                            class="px-4 py-2 rounded-lg border border-border hover:bg-bg-tertiary transition-colors"
                        >
                            取消
                        </button>
                        <button
                            @click="handleSave"
                            class="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover transition-colors font-medium"
                        >
                            保存设置
                        </button>
                    </div>
                </div>
            </div>
        </main>

        <!-- Provider 编辑对话框 -->
        <div
            v-if="showProviderDialog"
            class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
            @click.self="closeProviderDialog"
        >
            <div class="bg-bg-secondary rounded-xl border border-border w-full max-w-lg mx-4">
                <div class="p-4 border-b border-border">
                    <h2 class="text-lg font-medium">
                        {{ editingProvider ? "编辑 Provider" : "添加 Provider" }}
                    </h2>
                </div>
                <div class="p-4 space-y-4">
                    <div>
                        <label class="block text-sm text-text-secondary mb-2">Provider 名称</label>
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
                        <label for="provider-enabled" class="text-sm">
                            启用此 Provider
                        </label>
                    </div>
                    <div>
                        <label class="block text-sm text-text-secondary mb-2">API Key</label>
                        <input
                            v-model="providerForm.apiKey"
                            type="password"
                            placeholder="sk-..."
                            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                        />
                    </div>
                    <div>
                        <label class="block text-sm text-text-secondary mb-2">API Base URL</label>
                        <input
                            v-model="providerForm.apiBase"
                            type="text"
                            placeholder="https://api.openai.com/v1"
                            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                        />
                        <p class="text-xs text-text-secondary mt-1">可选，用于自定义 API 端点</p>
                    </div>
                    <div>
                        <label class="block text-sm text-text-secondary mb-2">默认模型</label>
                        <input
                            v-model="providerForm.model"
                            type="text"
                            placeholder="gpt-4, claude-3-opus, deepseek-chat"
                            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                        />
                    </div>
                </div>
                <div class="p-4 border-t border-border flex justify-end gap-3">
                    <button
                        @click="closeProviderDialog"
                        class="px-4 py-2 rounded-lg border border-border hover:bg-bg-tertiary transition-colors"
                    >
                        取消
                    </button>
                    <button
                        @click="handleSaveProvider"
                        :disabled="!providerForm.name"
                        class="px-4 py-2 bg-accent hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-medium transition-colors"
                    >
                        {{ savingProvider ? "保存中..." : "保存" }}
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted, computed, watch, reactive } from "vue";
import { useRouter } from "vue-router";
import {
    ArrowLeft as ArrowLeftIcon,
    Plus as PlusIcon,
    Edit as EditIcon,
    Trash as TrashIcon,
    Bot as BotIcon,
    Sparkles as SparklesIcon,
    ChevronRight as ChevronRightIcon,
    Moon as MoonIcon,
    Sun as SunIcon,
    Folder as FolderIcon,
    Palette as PaletteIcon,
    Info as InfoIcon,
    Wifi as ConnectionIcon,
} from "lucide-vue-next";

import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";
import { useThemeStore } from "@/stores/theme";
import { useSkillStore } from "@/stores/skill";
import api from "@/services/api";

const router = useRouter();
const chatStore = useChatStore();
const themeStore = useThemeStore();
const skillStore = useSkillStore();
const { status: wsStatus } = useWebSocket();

// 菜单项
const menuItems = [
    { key: 'connection', label: '连接设置', icon: ConnectionIcon },
    { key: 'workspace', label: '工作区', icon: FolderIcon },
    { key: 'provider', label: 'LLM Provider', icon: BotIcon },
    { key: 'skill', label: '技能管理', icon: SparklesIcon },
    { key: 'appearance', label: '外观', icon: PaletteIcon },
    { key: 'about', label: '关于', icon: InfoIcon },
];

// 当前选中
const activeSection = ref('connection');

// 表单数据
const wsUrl = ref(chatStore.wsUrl);
const apiBase = ref(chatStore.apiBase);
const userId = ref(chatStore.userId);

// 工作区
const workspace = ref("");
const savingWorkspace = ref(false);

// 配置文件
const configPath = ref("");
const configContent = ref("");
const savingConfig = ref(false);

// Provider 数据
const providers = ref([]);
const loading = ref(true);
const apiHealth = ref("checking");

// Provider 对话框
const showProviderDialog = ref(false);
const editingProvider = ref(null);
const savingProvider = ref(false);
const providerForm = reactive({
    name: "",
    enabled: true,
    apiKey: "",
    apiBase: "",
    model: ""
});

// 是否有修改
const hasChanges = computed(() => {
    return wsUrl.value !== chatStore.wsUrl ||
           apiBase.value !== chatStore.apiBase ||
           userId.value !== chatStore.userId;
});

// 获取 Provider 模型名称
function getProviderModel(provider) {
    try {
        const config = JSON.parse(provider.config || '{}');
        return config.model || '-';
    } catch {
        return '-';
    }
}

// 加载 Provider 列表
async function loadProviders() {
    loading.value = true;
    try {
        const response = await api.getProviders();
        providers.value = response.data || [];
    } catch (error) {
        console.error("获取 Provider 失败:", error);
        providers.value = [];
    }
    loading.value = false;
}

// 检查 API 健康状态
async function checkHealth() {
    try {
        await api.checkHealth();
        apiHealth.value = "ok";
    } catch (error) {
        apiHealth.value = "error";
    }
}

// 加载工作区
async function loadWorkspace() {
    try {
        const response = await api.getWorkspace();
        workspace.value = response.data?.workspace || "";
    } catch (error) {
        console.error("获取工作区失败:", error);
    }
}

// 设置工作区
async function handleSetWorkspace() {
    if (!workspace.value) return;

    savingWorkspace.value = true;
    try {
        await api.setWorkspace(workspace.value);
        alert("工作区已设置，需要重启服务才能生效");
    } catch (error) {
        console.error("设置工作区失败:", error);
        alert("设置工作区失败: " + error.message);
    }
    savingWorkspace.value = false;
}

// 加载配置文件
async function loadConfigFile() {
    try {
        const response = await api.getConfigFile();
        configPath.value = response.data?.path || "";
        configContent.value = response.data?.content || "";
    } catch (error) {
        console.error("获取配置文件失败:", error);
        configContent.value = "";
    }
}

// 覆盖配置文件
async function handleOverwriteConfig() {
    if (!configContent.value) return;

    savingConfig.value = true;
    try {
        await api.overwriteConfig(configContent.value);
        alert("配置文件已保存");
    } catch (error) {
        console.error("保存配置文件失败:", error);
        alert("保存配置文件失败: " + error.message);
    }
    savingConfig.value = false;
}

// 打开添加 Provider 对话框
function openAddProvider() {
    editingProvider.value = null;
    providerForm.name = "";
    providerForm.enabled = true;
    providerForm.apiKey = "";
    providerForm.apiBase = "";
    providerForm.model = "";
    showProviderDialog.value = true;
}

// 打开编辑 Provider 对话框
function openEditProvider(provider) {
    editingProvider.value = provider;
    providerForm.name = provider.name;
    providerForm.enabled = provider.enabled;
    
    // 解析 config
    try {
        const config = JSON.parse(provider.config || '{}');
        providerForm.apiKey = config.api_key || "";
        providerForm.apiBase = config.api_base || "";
        providerForm.model = config.model || "";
    } catch {
        providerForm.apiKey = "";
        providerForm.apiBase = "";
        providerForm.model = "";
    }
    
    showProviderDialog.value = true;
}

// 关闭 Provider 对话框
function closeProviderDialog() {
    showProviderDialog.value = false;
    editingProvider.value = null;
}

// 保存 Provider
async function handleSaveProvider() {
    if (!providerForm.name) return;

    savingProvider.value = true;
    
    const config = {
        api_key: providerForm.apiKey,
        api_base: providerForm.apiBase,
        model: providerForm.model
    };
    
    const data = {
        name: providerForm.name,
        enabled: providerForm.enabled,
        config: JSON.stringify(config)
    };

    try {
        if (editingProvider.value) {
            // 更新
            await api.updateProvider({
                id: editingProvider.value.id,
                ...data
            });
        } else {
            // 创建
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

// 重置表单
function handleReset() {
    wsUrl.value = chatStore.wsUrl;
    apiBase.value = chatStore.apiBase;
    userId.value = chatStore.userId;
}

// 保存设置
function handleSave() {
    chatStore.setWsUrl(wsUrl.value);
    chatStore.setApiBase(apiBase.value);
    chatStore.setUserId(userId.value);
    router.push("/");
}

// 监听菜单切换，重新加载对应数据
watch(activeSection, (newVal) => {
    if (newVal === 'workspace') {
        loadConfigFile();
    } else if (newVal === 'provider') {
        loadProviders();
    } else if (newVal === 'skill') {
        skillStore.fetchSkills();
    } else if (newVal === 'connection') {
        checkHealth();
    }
});

onMounted(() => {
    loadProviders();
    checkHealth();
    loadWorkspace();
    skillStore.fetchSkills();
});
</script>