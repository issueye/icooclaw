<template>
    <div class="flex w-full h-screen bg-[#0d0d0d] overflow-hidden">
        <!-- 左侧边栏 -->
        <ChatSidebar
            :sessions="chatStore.sessions"
            :current-session-id="chatStore.currentSessionId"
            :ws-status="wsStatus"
            :collapsed="sidebarCollapsed"
            @new="handleNewChat"
            @select="handleSelectSession"
            @delete="handleDeleteSession"
            @toggle="sidebarCollapsed = !sidebarCollapsed"
        />

        <!-- 主内容区 -->
        <div class="flex flex-col flex-1 min-w-0 h-full">
            <!-- 顶部 Header -->
            <ChatHeader
                :title="chatStore.currentSession?.title"
                :sidebar-collapsed="sidebarCollapsed"
                :api-status="apiStatus"
                @toggle-sidebar="sidebarCollapsed = !sidebarCollapsed"
                @new-chat="handleNewChat"
                @open-settings="router.push('/settings')"
            />

            <!-- 消息列表 -->
            <div
                ref="messagesContainer"
                class="flex-1 overflow-y-auto py-2"
                :class="
                    chatStore.currentMessages.length === 0
                        ? 'flex flex-col items-center justify-center'
                        : ''
                "
            >
                <!-- 欢迎空状态 -->
                <div
                    v-if="chatStore.currentMessages.length === 0"
                    class="text-center px-4 max-w-2xl"
                >
                    <div
                        class="w-16 h-16 mx-auto mb-6 rounded-2xl bg-gradient-to-br from-[#7c6af7] to-[#5b4fcf] flex items-center justify-center shadow-xl shadow-[#7c6af7]/20"
                    >
                        <BotIcon :size="28" class="text-white" />
                    </div>
                    <h2 class="text-2xl font-semibold text-[#f0f0f0] mb-3">
                        开始与 AI 对话
                    </h2>
                    <p class="text-[#909090] text-sm leading-relaxed mb-8">
                        icooclaw 是一个强大的 AI
                        Agent，支持工具调用、记忆系统和多种 LLM 模型。
                    </p>
                    <!-- 示例提示 -->
                    <div class="grid grid-cols-1 gap-2 text-left">
                        <button
                            v-for="hint in hints"
                            :key="hint"
                            @click="sendMessage(hint)"
                            class="px-4 py-3 rounded-xl bg-[#1e1e1e] border border-[#2a2a2a] text-sm text-[#909090] hover:bg-[#252525] hover:text-[#f0f0f0] hover:border-[#7c6af7]/30 transition-all text-left"
                        >
                            {{ hint }}
                        </button>
                    </div>
                </div>

                <!-- 消息列表 -->
                <div v-else class="w-full">
                    <ChatMessage
                        v-for="msg in chatStore.currentMessages"
                        :key="msg.id"
                        :message="msg"
                    />
                </div>
            </div>

            <!-- 输入区 -->
            <div class="w-full">
                <ChatInput
                    ref="chatInputRef"
                    :disabled="chatStore.isLoading"
                    @send="sendMessage"
                />
                <div class="flex items-center justify-between px-4 pb-2">
                    <p class="text-xs text-[#606060]">
                        连接到
                        <span class="text-[#7c6af7]">{{
                            chatStore.wsUrl
                        }}</span>
                    </p>
                    <p
                        class="text-xs"
                        :class="
                            apiStatus === 'ok'
                                ? 'text-green-500'
                                : 'text-red-500'
                        "
                    >
                        API: {{ apiStatus }}
                    </p>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { BotIcon } from "lucide-vue-next";

import ChatSidebar from "@/components/ChatSidebar.vue";
import ChatHeader from "@/components/ChatHeader.vue";
import ChatMessage from "@/components/ChatMessage.vue";
import ChatInput from "@/components/ChatInput.vue";

import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";
import api from "@/services/api";

const router = useRouter();
const chatStore = useChatStore();

// ===== WebSocket =====
const {
    status: wsStatus,
    connect,
    send,
    onMessage,
    disconnect,
} = useWebSocket();

function connectWs() {
    connect(chatStore.wsUrl);
}

// ===== API 状态 =====
const apiStatus = ref("checking");

async function checkApiStatus() {
    try {
        await api.checkHealth();
        apiStatus.value = "ok";
    } catch (error) {
        apiStatus.value = "error";
        console.error("API 健康检查失败:", error);
    }
}

// ===== 消息处理 =====
onMessage((msg) => {
    if (msg.type === "chunk") {
        chatStore.appendToLastAI(msg.content || "");
        scrollToBottom();
    } else if (msg.type === "chunk_end" || msg.type === "message") {
        if (msg.type === "message") {
            chatStore.finishLastAI(msg.content || "");
        } else {
            chatStore.finishLastAI();
        }
        chatStore.isLoading = false;
        scrollToBottom();
    } else if (msg.type === "error") {
        chatStore.finishLastAI("[错误] " + (msg.content || "未知错误"));
        chatStore.isLoading = false;
    }
});

async function sendMessage(text) {
    if (!text?.trim()) return;

    const session = chatStore.ensureSession();

    // 检查 WS 是否已连接
    if (wsStatus.value !== "connected") {
        connectWs();
        await new Promise((r) => setTimeout(r, 800));
    }

    chatStore.addUserMessage(text);
    scrollToBottom();

    chatStore.addAIMessage();
    chatStore.isLoading = true;
    scrollToBottom();

    const sent = send({
        type: "message",
        content: text,
        chat_id: session.chatId,
        user_id: chatStore.userId,
    });

    if (!sent) {
        chatStore.finishLastAI(
            "⚠️ 发送失败：WebSocket 未连接，请检查 Agent 服务是否启动",
        );
        chatStore.isLoading = false;
    }
}

// ===== 会话操作 =====
function handleNewChat() {
    chatStore.createSession();
    chatInputRef.value?.focus();
}

function handleSelectSession(id) {
    chatStore.switchSession(id);
    if (window.innerWidth < 768) {
        sidebarCollapsed.value = true;
    }
    scrollToBottom();
}

function handleDeleteSession(id) {
    chatStore.deleteSession(id);
}

// ===== UI 状态 =====
const sidebarCollapsed = ref(false);
const messagesContainer = ref(null);
const chatInputRef = ref(null);

const hints = [
    "你好，请介绍一下你自己",
    "帮我写一段 Python 快速排序代码",
    "今天天气怎么样？",
    "给我讲个有趣的笑话",
];

function scrollToBottom() {
    if (messagesContainer.value) {
        messagesContainer.value.scrollTop =
            messagesContainer.value.scrollHeight;
    }
}

watch(
    () => chatStore.currentMessages,
    () => scrollToBottom(),
    { deep: true },
);

// ===== 初始化 =====
onMounted(() => {
    if (window.innerWidth < 768) {
        sidebarCollapsed.value = true;
    }
    connectWs();
    checkApiStatus();
});
</script>
