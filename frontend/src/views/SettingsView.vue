<template>
    <div class="min-h-screen bg-[#0d0d0d] text-[#f0f0f0]">
        <!-- Header -->
        <header class="border-b border-[#2a2a2a] bg-[#151515]">
            <div
                class="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between"
            >
                <div class="flex items-center gap-3">
                    <button
                        @click="router.back()"
                        class="p-2 rounded-lg hover:bg-[#2a2a2a] transition-colors"
                    >
                        <ArrowLeftIcon :size="20" />
                    </button>
                    <h1 class="text-xl font-semibold">设置</h1>
                </div>
            </div>
        </header>

        <main class="max-w-4xl mx-auto px-4 py-6 space-y-6">
            <!-- 连接设置 -->
            <section
                class="bg-[#151515] rounded-xl border border-[#2a2a2a] p-6"
            >
                <h2 class="text-lg font-medium mb-4 flex items-center gap-2">
                    <SettingsIcon :size="20" class="text-[#7c6af7]" />
                    连接设置
                </h2>

                <div class="space-y-4">
                    <div>
                        <label class="block text-sm text-[#909090] mb-2"
                            >WebSocket 地址</label
                        >
                        <input
                            v-model="wsUrl"
                            type="text"
                            placeholder="ws://localhost:8080/ws"
                            class="w-full px-4 py-2.5 bg-[#1e1e1e] border border-[#2a2a2a] rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors"
                        />
                    </div>

                    <div>
                        <label class="block text-sm text-[#909090] mb-2"
                            >API 基础地址</label
                        >
                        <input
                            v-model="apiBase"
                            type="text"
                            placeholder="http://localhost:8080"
                            class="w-full px-4 py-2.5 bg-[#1e1e1e] border border-[#2a2a2a] rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors"
                        />
                    </div>

                    <div>
                        <label class="block text-sm text-[#909090] mb-2"
                            >用户 ID</label
                        >
                        <input
                            v-model="userId"
                            type="text"
                            placeholder="user-1"
                            class="w-full px-4 py-2.5 bg-[#1e1e1e] border border-[#2a2a2a] rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors"
                        />
                    </div>
                </div>
            </section>

            <!-- Provider 设置 -->
            <section
                class="bg-[#151515] rounded-xl border border-[#2a2a2a] p-6"
            >
                <h2 class="text-lg font-medium mb-4 flex items-center gap-2">
                    <BotIcon :size="20" class="text-[#7c6af7]" />
                    LLM Provider
                </h2>

                <div v-if="loading" class="text-[#909090]">加载中...</div>

                <div
                    v-else-if="providers.length > 0"
                    class="grid grid-cols-2 md:grid-cols-3 gap-3"
                >
                    <div
                        v-for="provider in providers"
                        :key="provider.name"
                        class="p-3 bg-[#1e1e1e] rounded-lg border border-[#2a2a2a]"
                    >
                        <div class="font-medium text-sm">
                            {{ provider.name }}
                        </div>
                        <div class="text-xs text-[#909090] mt-1">
                            {{ provider.model }}
                        </div>
                    </div>
                </div>

                <div v-else class="text-[#909090] text-sm">
                    无法获取 Provider 信息，请检查后端服务是否运行
                </div>
            </section>

            <!-- 技能管理入口 -->
            <section
                @click="router.push('/skills')"
                class="bg-[#151515] rounded-xl border border-[#2a2a2a] p-6 cursor-pointer hover:border-[#7c6af7]/50 transition-colors"
            >
                <h2 class="text-lg font-medium mb-4 flex items-center gap-2">
                    <SparklesIcon :size="20" class="text-[#7c6af7]" />
                    技能管理
                    <ChevronRightIcon
                        :size="20"
                        class="ml-auto text-[#909090]"
                    />
                </h2>
                <p class="text-sm text-[#909090]">
                    管理自定义技能，扩展 AI 助手能力
                </p>
            </section>

            <!-- 状态信息 -->
            <section
                class="bg-[#151515] rounded-xl border border-[#2a2a2a] p-6"
            >
                <h2 class="text-lg font-medium mb-4 flex items-center gap-2">
                    <ActivityIcon :size="20" class="text-[#7c6af7]" />
                    连接状态
                </h2>

                <div class="space-y-3">
                    <div class="flex items-center justify-between">
                        <span class="text-[#909090]">API 状态</span>
                        <span
                            :class="
                                apiHealth === 'ok'
                                    ? 'text-green-500'
                                    : 'text-red-500'
                            "
                        >
                            {{ apiHealth === "ok" ? "已连接" : "未连接" }}
                        </span>
                    </div>
                    <div class="flex items-center justify-between">
                        <span class="text-[#909090]">WebSocket</span>
                        <span class="text-[#909090]">{{ wsStatus }}</span>
                    </div>
                    <div class="flex items-center justify-between">
                        <span class="text-[#909090]">会话数量</span>
                        <span class="text-[#909090]">{{ sessionCount }}</span>
                    </div>
                </div>
            </section>

            <!-- 保存按钮 -->
            <div class="flex justify-end gap-3">
                <button
                    @click="router.back()"
                    class="px-6 py-2.5 rounded-lg border border-[#2a2a2a] hover:bg-[#1e1e1e] transition-colors"
                >
                    取消
                </button>
                <button
                    @click="handleSave"
                    class="px-6 py-2.5 rounded-lg bg-[#7c6af7] hover:bg-[#6b5ce7] transition-colors font-medium"
                >
                    保存设置
                </button>
            </div>
        </main>
    </div>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import {
    ArrowLeftIcon,
    Settings as SettingsIcon,
    Bot as BotIcon,
    Activity as ActivityIcon,
    Sparkles as SparklesIcon,
    ChevronRight as ChevronRightIcon,
} from "lucide-vue-next";

import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";
import api from "@/services/api";

const router = useRouter();
const chatStore = useChatStore();
const { status: wsStatus } = useWebSocket();

// 表单数据
const wsUrl = ref(chatStore.wsUrl);
const apiBase = ref(chatStore.apiBase);
const userId = ref(chatStore.userId);

// Provider 数据
const providers = ref([]);
const loading = ref(true);
const apiHealth = ref("checking");

const sessionCount = computed(() => chatStore.sessions.length);

// 加载 Provider 列表
async function loadProviders() {
    loading.value = true;
    try {
        const data = await api.getProviders();
        providers.value = data.providers || [];
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

// 保存设置
function handleSave() {
    chatStore.setWsUrl(wsUrl.value);
    chatStore.setApiBase(apiBase.value);
    chatStore.setUserId(userId.value);

    // 重新连接 WebSocket
    router.push("/");
}

onMounted(() => {
    loadProviders();
    checkHealth();
});
</script>
