<template>
  <section class="space-y-6">
    <div>
      <h2 class="text-xl font-semibold mb-1">连接设置</h2>
      <p class="text-text-secondary text-sm">
        配置 API 和 WebSocket 连接地址
      </p>
    </div>

    <div class="bg-bg-secondary rounded-xl border border-border p-6 space-y-4">
      <div class="grid grid-cols-3 gap-3">
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            服务器 IP
          </label>
          <input
            v-model="localWsHost"
            type="text"
            placeholder="localhost"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            端口
          </label>
          <input
            v-model="localWsPort"
            type="text"
            placeholder="8080"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            路径
          </label>
          <input
            v-model="localWsPath"
            type="text"
            placeholder="/ws/chat"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>
      </div>

      <div>
        <label class="block text-sm text-text-secondary mb-2">
          API 基础地址
        </label>
        <input
          v-model="localApiBase"
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
          v-model="localUserId"
          type="text"
          placeholder="user-1"
          class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
        />
      </div>

      <div class="flex gap-3 pt-2">
        <button
          v-if="!wsConnected"
          @click="handleConnect"
          :disabled="connecting"
          class="flex-1 px-4 py-2.5 bg-green-600 hover:bg-green-700 disabled:opacity-50 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
        >
          <WifiIcon v-if="!connecting" :size="16" />
          <Loader2Icon v-else :size="16" class="animate-spin" />
          {{ connecting ? "连接中..." : "连接" }}
        </button>
        <button
          v-else
          @click="handleDisconnect"
          class="flex-1 px-4 py-2.5 bg-red-600 hover:bg-red-700 rounded-lg text-sm font-medium transition-colors flex items-center justify-center gap-2"
        >
          <WifiOffIcon :size="16" />
          断开连接
        </button>
        <button
          @click="handleSave"
          class="px-4 py-2.5 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
        >
          保存设置
        </button>
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
              apiHealth === 'ok' ? 'text-green-500' : 'text-red-500',
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
</template>

<script setup>
import { ref, watch, onMounted } from "vue";
import {
  Wifi as WifiIcon,
  WifiOff as WifiOffIcon,
  Loader2 as Loader2Icon,
} from "lucide-vue-next";
import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";
import api from "@/services/api";

const emit = defineEmits(["connect", "disconnect", "save"]);

const chatStore = useChatStore();
const { status: wsStatus } = useWebSocket();

// 本地状态
const localWsHost = ref(chatStore.wsHost);
const localWsPort = ref(chatStore.wsPort);
const localWsPath = ref(localStorage.getItem("icooclaw_ws_path") || "/ws/chat");
const localApiBase = ref(chatStore.apiBase);
const localUserId = ref(chatStore.userId);

// 连接状态
const wsConnected = ref(chatStore.wsConnected);
const connecting = ref(false);
const apiHealth = ref("checking");

// 检查 API 健康状态
async function checkHealth() {
  try {
    await api.checkHealth();
    apiHealth.value = "ok";
  } catch (error) {
    apiHealth.value = "error";
  }
}

// 连接
async function handleConnect() {
  connecting.value = true;
  saveToStore();
  emit("connect");
  setTimeout(() => {
    connecting.value = false;
    wsConnected.value = chatStore.wsConnected;
  }, 1000);
}

// 断开连接
function handleDisconnect() {
  emit("disconnect");
  wsConnected.value = false;
}

// 保存设置
function handleSave() {
  saveToStore();
  emit("save");
}

// 保存到 store
function saveToStore() {
  chatStore.setWsHost(localWsHost.value);
  chatStore.setWsPort(localWsPort.value);
  localStorage.setItem("icooclaw_ws_path", localWsPath.value);
  chatStore.setApiBase(localApiBase.value);
  chatStore.setUserId(localUserId.value);
}

// 监听 store 变化
watch(
  () => chatStore.wsConnected,
  (val) => {
    wsConnected.value = val;
  }
);

onMounted(() => {
  checkHealth();
});
</script>