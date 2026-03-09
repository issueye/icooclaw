<template>
  <div class="flex w-full h-screen bg-bg-primary overflow-hidden">
    <!-- 左侧边栏 -->
    <div
      class="w-64 bg-bg-secondary border-r border-border flex flex-col"
      :class="collapsed ? 'w-16' : ''"
    >
      <!-- 标题 -->
      <div class="p-4 border-b border-border flex items-center justify-between">
        <h1 v-if="!collapsed" class="text-lg font-semibold text-text-primary">
          Agent 管理
        </h1>
        <button
          @click="collapsed = !collapsed"
          class="p-1 rounded hover:bg-bg-hover text-text-secondary"
        >
          <ChevronLeftIcon v-if="!collapsed" :size="20" />
          <ChevronRightIcon v-else :size="20" />
        </button>
      </div>

      <!-- 连接状态 -->
      <div class="p-4 border-b border-border">
        <div class="flex items-center gap-2 mb-3">
          <span
            class="w-2 h-2 rounded-full"
            :class="acpStore.connected ? 'bg-green-500' : 'bg-red-500'"
          ></span>
          <span class="text-sm text-text-secondary">
            {{ acpStore.connected ? '已连接' : '未连接' }}
          </span>
        </div>
        <button
          v-if="!acpStore.connected"
          @click="handleConnect"
          :disabled="acpStore.connecting"
          class="w-full px-3 py-2 bg-accent text-white rounded-lg text-sm hover:bg-accent/90 disabled:opacity-50"
        >
          {{ acpStore.connecting ? '连接中...' : '连接 AP' }}
        </button>
        <button
          v-else
          @click="handleDisconnect"
          class="w-full px-3 py-2 bg-bg-tertiary text-text-secondary rounded-lg text-sm hover:bg-bg-hover"
        >
          断开连接
        </button>
      </div>

      <!-- Agent 列表 -->
      <div class="flex-1 overflow-y-auto p-2">
        <div v-if="!collapsed" class="text-xs text-text-muted px-2 mb-2">
          已连接 Agent ({{ acpStore.agents.length }})
        </div>
        <div
          v-for="agent in acpStore.agents"
          :key="agent.aid"
          @click="selectAgent(agent)"
          class="p-2 rounded-lg cursor-pointer mb-1 transition-colors"
          :class="
            acpStore.currentAgent?.aid === agent.aid
              ? 'bg-accent/20 border border-accent/30'
              : 'hover:bg-bg-hover'
          "
        >
          <div v-if="!collapsed" class="flex items-center gap-2">
            <BotIcon :size="16" class="text-accent" />
            <span class="text-sm text-text-primary truncate">{{
              agent.profile?.name || agent.aid
            }}</span>
          </div>
          <div v-else class="flex justify-center">
            <BotIcon :size="16" class="text-accent" />
          </div>
        </div>
      </div>

      <!-- 底部导航 -->
      <div class="p-2 border-t border-border">
        <button
          @click="router.push('/')"
          class="w-full p-2 rounded-lg flex items-center gap-2 hover:bg-bg-hover text-text-secondary"
          :class="route.path === '/' ? 'bg-bg-hover text-accent' : ''"
        >
          <MessageCircleIcon :size="18" />
          <span v-if="!collapsed" class="text-sm">聊天</span>
        </button>
        <button
          @click="router.push('/settings')"
          class="w-full p-2 rounded-lg flex items-center gap-2 hover:bg-bg-hover text-text-secondary"
          :class="route.path === '/settings' ? 'bg-bg-hover text-accent' : ''"
        >
          <SettingsIcon :size="18" />
          <span v-if="!collapsed" class="text-sm">设置</span>
        </button>
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- 顶部 Header -->
      <div class="h-14 px-4 border-b border-border flex items-center justify-between bg-bg-secondary">
        <div class="flex items-center gap-3">
          <h2 class="text-lg font-semibold text-text-primary">
            {{ acpStore.currentAgent?.profile?.name || '选择 Agent' }}
          </h2>
          <span
            v-if="acpStore.currentAgent"
            class="px-2 py-0.5 text-xs rounded-full"
            :class="
              acpStore.currentAgent.status === 'connected'
                ? 'bg-green-500/20 text-green-500'
                : 'bg-gray-500/20 text-gray-500'
            "
          >
            {{ acpStore.currentAgent.status }}
          </span>
        </div>
        <div class="flex items-center gap-2">
          <button
            v-if="acpStore.currentAgent"
            @click="startChat"
            class="px-3 py-1.5 bg-accent text-white rounded-lg text-sm hover:bg-accent/90"
          >
            开始聊天
          </button>
        </div>
      </div>

      <!-- Agent 详情 -->
      <div v-if="acpStore.currentAgent" class="flex-1 overflow-y-auto p-4">
        <div class="max-w-3xl">
          <!-- 基本信息 -->
          <div class="bg-bg-secondary rounded-xl p-4 mb-4">
            <h3 class="text-sm font-medium text-text-primary mb-3">基本信息</h3>
            <div class="space-y-2 text-sm">
              <div class="flex">
                <span class="w-20 text-text-muted">AID:</span>
                <span class="text-text-primary font-mono">{{
                  acpStore.currentAgent.aid
                }}</span>
              </div>
              <div class="flex">
                <span class="w-20 text-text-muted">版本:</span>
                <span class="text-text-primary">{{
                  acpStore.currentAgent.profile?.version || 'N/A'
                }}</span>
              </div>
              <div class="flex">
                <span class="w-20 text-text-muted">发布者:</span>
                <span class="text-text-primary">{{
                  acpStore.currentAgent.profile?.publisherInfo || 'N/A'
                }}</span>
              </div>
            </div>
          </div>

          <!-- 描述 -->
          <div
            v-if="acpStore.currentAgent.profile?.description"
            class="bg-bg-secondary rounded-xl p-4 mb-4"
          >
            <h3 class="text-sm font-medium text-text-primary mb-2">描述</h3>
            <p class="text-sm text-text-secondary">
              {{ acpStore.currentAgent.profile.description }}
            </p>
          </div>

          <!-- 能力 -->
          <div
            v-if="acpStore.currentAgent.profile?.capabilities"
            class="bg-bg-secondary rounded-xl p-4 mb-4"
          >
            <h3 class="text-sm font-medium text-text-primary mb-3">能力</h3>
            <div class="space-y-3">
              <div v-if="acpStore.currentAgent.profile.capabilities.core?.length">
                <span class="text-xs text-text-muted">核心能力:</span>
                <div class="flex flex-wrap gap-1 mt-1">
                  <span
                    v-for="cap in acpStore.currentAgent.profile.capabilities.core"
                    :key="cap"
                    class="px-2 py-0.5 text-xs bg-accent/20 text-accent rounded"
                  >
                    {{ cap }}
                  </span>
                </div>
              </div>
              <div
                v-if="acpStore.currentAgent.profile.capabilities.extended?.length"
              >
                <span class="text-xs text-text-muted">扩展能力:</span>
                <div class="flex flex-wrap gap-1 mt-1">
                  <span
                    v-for="cap in acpStore.currentAgent.profile.capabilities.extended"
                    :key="cap"
                    class="px-2 py-0.5 text-xs bg-bg-tertiary text-text-secondary rounded"
                  >
                    {{ cap }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- 输入输出 -->
          <div
            v-if="acpStore.currentAgent.profile?.input || acpStore.currentAgent.profile?.output"
            class="bg-bg-secondary rounded-xl p-4"
          >
            <h3 class="text-sm font-medium text-text-primary mb-3">输入输出</h3>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <span class="text-xs text-text-muted">支持输入:</span>
                <div class="flex flex-wrap gap-1 mt-1">
                  <span
                    v-for="t in acpStore.currentAgent.profile?.input?.types"
                    :key="t"
                    class="px-2 py-0.5 text-xs bg-bg-tertiary text-text-secondary rounded"
                  >
                    {{ t }}
                  </span>
                </div>
              </div>
              <div>
                <span class="text-xs text-text-muted">输出类型:</span>
                <div class="flex flex-wrap gap-1 mt-1">
                  <span
                    v-for="t in acpStore.currentAgent.profile?.output?.types"
                    :key="t"
                    class="px-2 py-0.5 text-xs bg-bg-tertiary text-text-secondary rounded"
                  >
                    {{ t }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="flex-1 flex items-center justify-center">
        <div class="text-center">
          <div
            class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-bg-tertiary flex items-center justify-center"
          >
            <BotIcon :size="28" class="text-text-muted" />
          </div>
          <h3 class="text-lg font-medium text-text-primary mb-2">
            连接到 Agent
          </h3>
          <p class="text-sm text-text-secondary mb-4">
            从左侧选择一个已连接的 Agent 查看详情
          </p>
          <p class="text-xs text-text-muted">
            提示: 请先连接到 AP 接入点
          </p>
        </div>
      </div>
    </div>

    <!-- 配置对话框 -->
    <div
      v-if="showConfigDialog"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="showConfigDialog = false"
    >
      <div class="bg-bg-secondary rounded-xl p-6 w-full max-w-md">
        <h3 class="text-lg font-semibold text-text-primary mb-4">AP 配置</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm text-text-secondary mb-1">
              接入点地址
            </label>
            <input
              v-model="configForm.endpoint"
              type="text"
              class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-text-primary"
              placeholder="wss://ap.agentunion.cn"
            />
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-1">
              API Key
            </label>
            <input
              v-model="configForm.apiKey"
              type="password"
              class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-text-primary"
              placeholder="可选"
            />
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-1">
              本地 AID
            </label>
            <input
              v-model="configForm.aid"
              type="text"
              class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-text-primary"
              placeholder="可选"
            />
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-6">
          <button
            @click="showConfigDialog = false"
            class="px-4 py-2 text-text-secondary hover:bg-bg-hover rounded-lg"
          >
            取消
          </button>
          <button
            @click="saveConfig"
            class="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent/90"
          >
            保存
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import {
  BotIcon,
  MessageCircleIcon,
  SettingsIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from 'lucide-vue-next';
import { useACPStore } from '@/stores/acp';

const router = useRouter();
const route = useRoute();
const acpStore = useACPStore();

const collapsed = ref(false);
const showConfigDialog = ref(false);
const configForm = reactive({
  endpoint: '',
  apiKey: '',
  aid: '',
});

onMounted(async () => {
  await acpStore.init();
  Object.assign(configForm, acpStore.config);
});

function handleConnect() {
  acpStore.updateConfig(configForm);
  acpStore.connect().catch((err) => {
    console.error('连接失败:', err);
  });
}

function handleDisconnect() {
  acpStore.disconnect();
}

function selectAgent(agent) {
  acpStore.currentAgent = agent;
}

function startChat() {
  if (acpStore.currentAgent) {
    router.push('/');
  }
}

function saveConfig() {
  acpStore.updateConfig(configForm);
  showConfigDialog = false;
}
</script>