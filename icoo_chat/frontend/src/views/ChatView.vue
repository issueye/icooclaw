<template>
    <div class="flex w-full h-screen bg-bg-primary overflow-hidden">
        <!-- 模式切换 -->
        <div
            class="fixed top-4 right-4 z-50 flex items-center gap-2 bg-bg-secondary rounded-lg p-1 border border-border"
        >
            <button
                @click="chatMode = 'ws'"
                class="px-3 py-1.5 rounded text-sm transition-colors"
                :class="
                    chatMode === 'ws'
                        ? 'bg-accent text-white'
                        : 'text-text-secondary hover:bg-bg-hover'
                "
            >
                WebSocket
            </button>
            <button
                @click="chatMode = 'acp'"
                class="px-3 py-1.5 rounded text-sm transition-colors"
                :class="
                    chatMode === 'acp'
                        ? 'bg-accent text-white'
                        : 'text-text-secondary hover:bg-bg-hover'
                "
            >
                ACP
            </button>
        </div>

        <!-- ACP 模式 -->
        <template v-if="chatMode === 'acp'">
            <!-- 左侧边栏 - Agent 列表 -->
            <div
                class="w-64 bg-bg-secondary border-r border-border flex flex-col"
                :class="sidebarCollapsed ? 'w-16' : ''"
            >
                <div class="p-4 border-b border-border flex items-center justify-between">
                    <h1 v-if="!sidebarCollapsed" class="text-lg font-semibold text-text-primary">
                        Agent 列表
                    </h1>
                    <button
                        @click="sidebarCollapsed = !sidebarCollapsed"
                        class="p-1 rounded hover:bg-bg-hover text-text-secondary"
                    >
                        <ChevronLeftIcon v-if="!sidebarCollapsed" :size="20" />
                        <ChevronRightIcon v-else :size="20" />
                    </button>
                </div>

                <!-- 连接状态 -->
                <div class="p-4 border-b border-border">
                    <div class="flex items-center gap-2 mb-2">
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
                        @click="initACP"
                        class="w-full px-3 py-2 bg-accent text-white rounded-lg text-sm hover:bg-accent/90"
                    >
                        连接 AP
                    </button>
                    <div v-else class="text-xs text-text-muted">
                        {{ acpStore.config.endpoint }}
                    </div>
                </div>

                <!-- Agent 列表 -->
                <div class="flex-1 overflow-y-auto p-2">
                    <div
                        v-for="agent in acpStore.agents"
                        :key="agent.aid"
                        @click="selectACPAgent(agent)"
                        class="p-2 rounded-lg cursor-pointer mb-1 transition-colors"
                        :class="
                            acpStore.currentAgent?.aid === agent.aid
                                ? 'bg-accent/20 border border-accent/30'
                                : 'hover:bg-bg-hover'
                        "
                    >
                        <div class="flex items-center gap-2">
                            <BotIcon :size="16" class="text-accent" />
                            <span class="text-sm text-text-primary truncate">{{
                                agent.profile?.name || agent.aid
                            }}</span>
                        </div>
                    </div>
                </div>

                <!-- 底部导航 -->
                <div class="p-2 border-t border-border">
                    <button
                        @click="router.push('/agents')"
                        class="w-full p-2 rounded-lg flex items-center gap-2 hover:bg-bg-hover text-text-secondary"
                    >
                        <UsersIcon :size="18" />
                        <span v-if="!sidebarCollapsed" class="text-sm">管理 Agent</span>
                    </button>
                    <button
                        @click="router.push('/settings')"
                        class="w-full p-2 rounded-lg flex items-center gap-2 hover:bg-bg-hover text-text-secondary"
                    >
                        <SettingsIcon :size="18" />
                        <span v-if="!sidebarCollapsed" class="text-sm">设置</span>
                    </button>
                </div>
            </div>

            <!-- 主内容区 -->
            <div class="flex flex-col flex-1 min-w-0 h-full">
                <!-- 顶部 Header -->
                <div class="h-14 px-4 border-b border-border flex items-center justify-between bg-bg-secondary">
                    <div class="flex items-center gap-3">
                        <h2 class="text-lg font-semibold text-text-primary">
                            {{ acpStore.currentAgent?.profile?.name || '选择 Agent 开始对话' }}
                        </h2>
                        <span
                            v-if="acpStore.currentSession"
                            class="px-2 py-0.5 text-xs bg-green-500/20 text-green-500 rounded-full"
                        >
                            会话中
                        </span>
                    </div>
                    <div class="flex items-center gap-2">
                        <button
                            v-if="acpStore.currentSession"
                            @click="closeACPSession"
                            class="px-3 py-1.5 text-text-secondary hover:bg-bg-hover rounded-lg text-sm"
                        >
                            结束会话
                        </button>
                    </div>
                </div>

                <!-- 消息列表 -->
                <div
                    ref="messagesContainer"
                    class="flex-1 overflow-y-auto py-2"
                    :class="
                        acpStore.messages.length === 0
                            ? 'flex flex-col items-center justify-center'
                            : ''
                    "
                >
                    <!-- 欢迎空状态 -->
                    <div
                        v-if="acpStore.messages.length === 0"
                        class="text-center px-4 max-w-2xl"
                    >
                        <div
                            class="w-16 h-16 mx-auto mb-6 rounded-2xl bg-linear-to-br from-accent to-[#5b4fcf] flex items-center justify-center shadow-xl shadow-[#7c6af7]/20"
                        >
                            <BotIcon :size="28" class="text-white" />
                        </div>
                        <h2 class="text-2xl font-semibold text-text-primary mb-3">
                            {{ acpStore.currentAgent ? '开始对话' : '选择 Agent' }}
                        </h2>
                        <p class="text-text-secondary text-sm leading-relaxed mb-8">
                            {{
                                acpStore.currentAgent
                                    ? `与 ${acpStore.currentAgent.profile?.name} 开始对话`
                                    : '请从左侧选择一个已连接的 Agent'
                            }}
                        </p>
                        <div v-if="acpStore.currentAgent" class="grid grid-cols-1 gap-2 text-left">
                            <button
                                v-for="hint in hints"
                                :key="hint"
                                @click="sendACPMessage(hint)"
                                class="px-4 py-3 rounded-xl bg-bg-tertiary border border-border text-sm text-text-secondary hover:bg-bg-hover hover:text-text-primary hover:border-accent/30 transition-all text-left"
                            >
                                {{ hint }}
                            </button>
                        </div>
                    </div>

                    <!-- 消息列表 -->
                    <div v-else class="w-full">
                        <div
                            v-for="msg in acpStore.messages"
                            :key="msg.id"
                            class="mb-4"
                        >
                            <!-- 用户消息 -->
                            <div v-if="msg.role === 'user'" class="flex justify-end px-4">
                                <div class="max-w-[70%] px-4 py-2 bg-accent text-white rounded-2xl">
                                    {{ msg.content }}
                                </div>
                            </div>

                            <!-- AI 消息 -->
                            <div v-else class="px-4">
                                <div class="flex gap-3">
                                    <div class="w-8 h-8 rounded-full bg-accent flex items-center justify-center flex-shrink-0">
                                        <BotIcon :size="16" class="text-white" />
                                    </div>
                                    <div class="flex-1 min-w-0">
                                        <!-- 思考过程 -->
                                        <div
                                            v-if="msg.thinking"
                                            class="text-xs text-text-muted mb-2 italic"
                                        >
                                            {{ msg.thinking }}
                                        </div>
                                        <!-- 消息内容 -->
                                        <div class="text-text-primary whitespace-pre-wrap">
                                            {{ msg.content }}
                                        </div>
                                        <!-- 工具调用 -->
                                        <ToolCallDisplay
                                            v-if="msg.toolCalls?.length"
                                            :tool-calls="msg.toolCalls"
                                        />
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 输入区 -->
                <div class="w-full">
                    <div class="flex gap-2 px-4 py-2">
                        <input
                            v-model="acpInput"
                            @keyup.enter="sendACPMessage(acpInput)"
                            :disabled="!acpStore.currentAgent || acpStore.loading"
                            class="flex-1 px-4 py-2 bg-bg-tertiary border border-border rounded-lg text-text-primary placeholder-text-muted"
                            placeholder="输入消息..."
                        />
                        <button
                            @click="sendACPMessage(acpInput)"
                            :disabled="!acpStore.currentAgent || acpStore.loading || !acpInput.trim()"
                            class="px-4 py-2 bg-accent text-white rounded-lg hover:bg-accent/90 disabled:opacity-50"
                        >
                            <SendIcon :size="18" />
                        </button>
                    </div>
                    <div class="flex items-center justify-between px-4 pb-2">
                        <p class="text-xs text-text-muted">
                            模式: ACP
                            <span v-if="acpStore.currentAgent">
                                | Agent: {{ acpStore.currentAgent.profile?.name }}
                            </span>
                        </p>
                        <p
                            class="text-xs"
                            :class="acpStore.connected ? 'text-green-500' : 'text-red-500'"
                        >
                            {{ acpStore.connected ? '已连接' : '未连接' }}
                        </p>
                    </div>
                </div>
            </div>
        </template>

        <!-- WebSocket 模式 -->
        <template v-else>
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
                            class="w-16 h-16 mx-auto mb-6 rounded-2xl bg-linear-to-br from-accent to-[#5b4fcf] flex items-center justify-center shadow-xl shadow-[#7c6af7]/20"
                        >
                            <BotIcon :size="28" class="text-white" />
                        </div>
                        <h2 class="text-2xl font-semibold text-text-primary mb-3">
                            开始与 AI 对话
                        </h2>
                        <p class="text-text-secondary text-sm leading-relaxed mb-8">
                            icooclaw 是一个强大的 AI
                            Agent，支持工具调用、记忆系统和多种 LLM 模型。
                        </p>
                        <!-- 示例提示 -->
                        <div class="grid grid-cols-1 gap-2 text-left">
                            <button
                                v-for="hint in hints"
                                :key="hint"
                                @click="sendMessage(hint)"
                                class="px-4 py-3 rounded-xl bg-bg-tertiary border border-border text-sm text-text-secondary hover:bg-bg-hover hover:text-text-primary hover:border-accent/30 transition-all text-left"
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
                        <p class="text-xs text-text-muted">
                            连接到
                            <span class="text-accent">{{ chatStore.wsUrl }}</span>
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
        </template>
    </div>
</template>

<script setup>
import { ref, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import {
    BotIcon,
    ChevronLeftIcon,
    ChevronRightIcon,
    SettingsIcon,
    UsersIcon,
    SendIcon,
} from "lucide-vue-next";

import ChatSidebar from "@/components/ChatSidebar.vue";
import ChatHeader from "@/components/ChatHeader.vue";
import ChatMessage from "@/components/ChatMessage.vue";
import ChatInput from "@/components/ChatInput.vue";
import ToolCallDisplay from "@/components/ToolCallDisplay.vue";

import { useChatStore } from "@/stores/chat";
import { useACPStore } from "@/stores/acp";
import { useWebSocket } from "@/composables/useWebSocket";
import api from "@/services/api";

const router = useRouter();
const chatStore = useChatStore();
const acpStore = useACPStore();

// ===== 聊天模式 =====
const chatMode = ref("ws");
const acpInput = ref("");

// ===== ACP 功能 =====
async function initACP() {
    try {
        await acpStore.connect();
    } catch (error) {
        console.error("连接 ACP 失败:", error);
    }
}

function selectACPAgent(agent) {
    acpStore.currentAgent = agent;
    acpStore.messages = [];
}

async function sendACPMessage(text) {
    if (!text?.trim() || !acpStore.currentAgent) return;

    try {
        await acpStore.sendMessage(text);
        scrollToBottom();
    } catch (error) {
        console.error("发送消息失败:", error);
    }
}

async function closeACPSession() {
    await acpStore.closeSession();
}

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
    // 处理后端返回的消息
    switch (msg.type) {
        case "session_created":
            // 会话创建成功，保存 session_id
            if (msg.data && msg.data.session_id) {
                chatStore.setWsSessionId(
                    chatStore.currentSessionId,
                    msg.data.session_id
                );
                // 发送聊天消息
                sendPendingChatMessage();
            }
            break;
        case "chunk":
            chatStore.appendToLastAI(msg.data?.content || "");
            scrollToBottom();
            break;
        case "thinking":
            chatStore.updateThinking(cleanThinkingContent(msg.data?.content || ""));
            break;
        case "tool_call":
            chatStore.addToolCall({
                id: msg.data?.tool_call_id,
                tool_name: msg.data?.tool_name,
                arguments: msg.data?.arguments,
                status: "pending",
                timestamp: Date.now(),
            });
            scrollToBottom();
            break;
        case "tool_result":
            chatStore.updateToolResult({
                id: msg.data?.tool_call_id,
                status: "completed",
                content: msg.data?.result,
            });
            scrollToBottom();
            break;
        case "end":
            chatStore.finishLastAI();
            chatStore.isLoading = false;
            scrollToBottom();
            break;
        case "error":
            chatStore.finishLastAI(
                "[错误] " + (msg.error?.message || "未知错误")
            );
            chatStore.isLoading = false;
            break;
        case "queue_status":
            // 队列状态更新，可以显示给用户
            console.log("队列状态:", msg.data);
            break;
        case "pong":
            // 心跳响应
            break;
        default:
            console.log("未处理的消息类型:", msg.type, msg);
    }
});

// 待发送的聊天消息
let pendingChatContent = null;

function sendPendingChatMessage() {
    if (!pendingChatContent) return;
    
    const wsSessionId = chatStore.getWsSessionId(chatStore.currentSessionId);
    if (!wsSessionId) {
        console.error("没有有效的 WebSocket 会话ID");
        chatStore.finishLastAI("⚠️ 创建会话失败：无法获取会话ID");
        chatStore.isLoading = false;
        pendingChatContent = null;
        return;
    }

    const sent = send({
        type: "chat",
        session_id: wsSessionId,
        data: {
            session_id: wsSessionId,
            content: pendingChatContent,
        },
    });

    if (!sent) {
        chatStore.finishLastAI(
            "⚠️ 发送失败：WebSocket 连接缓冲区已满，请稍后重试"
        );
        chatStore.isLoading = false;
    }
    
    pendingChatContent = null;
}

async function sendMessage(text) {
    if (!text?.trim()) return;

    const session = chatStore.ensureSession();

    // 检查 WS 是否已连接，如未连接则等待连接成功
    if (wsStatus.value !== "connected" && wsStatus.value !== "reconnecting") {
        connectWs();
        
        // 轮询等待连接成功，最多等待 10 秒
        const maxWaitTime = 10000;
        const checkInterval = 200;
        let waitedTime = 0;
        
        while (wsStatus.value !== "connected" && waitedTime < maxWaitTime) {
            await new Promise(r => setTimeout(r, checkInterval));
            waitedTime += checkInterval;
        }
        
        if (wsStatus.value !== "connected") {
            chatStore.addUserMessage(text);
            chatStore.addAIMessage();
            chatStore.finishLastAI(
                "⚠️ 连接失败：请检查 Agent 服务是否启动"
            );
            return;
        }
    }

    chatStore.addUserMessage(text);
    scrollToBottom();

    chatStore.addAIMessage();
    chatStore.isLoading = true;
    scrollToBottom();

    // 检查是否已有 WebSocket 会话
    const wsSessionId = chatStore.getWsSessionId(chatStore.currentSessionId);
    
    if (wsSessionId) {
        // 已有会话，直接发送聊天消息
        const sent = send({
            type: "chat",
            session_id: wsSessionId,
            data: {
                session_id: wsSessionId,  // 后端需要这个字段进行校验
                content: text,
            },
        });

        if (!sent) {
            chatStore.finishLastAI(
                "⚠️ 发送失败：WebSocket 连接缓冲区已满，请稍后重试"
            );
            chatStore.isLoading = false;
        }
    } else {
        // 需要先创建 WebSocket 会话
        pendingChatContent = text;
        // 传递前端已有的 session_id，让后端复用该会话
        send({
            type: "create_session",
            data: {
                session_id: chatStore.currentSessionId,
                channel: "websocket",
                user_id: chatStore.userId,
            },
        });
    }
}

// ===== 会话操作 =====
function handleNewChat() {
    chatStore.createSession();
    chatInputRef.value?.focus();
}

async function handleSelectSession(id) {
    console.log('切换会话:', id);
    
    await chatStore.switchSession(id);
    if (window.innerWidth < 768) {
        sidebarCollapsed.value = true;
    }
    scrollToBottom();
}

function handleDeleteSession(id) {
    console.log('删除会话:', id);
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

// 清理思考内容中的模型特定标签
function cleanThinkingContent(content) {
    if (!content) return "";
    return content
        .replace(/<think>/g, "")
        .replace(/<\/think>/g, "")
        .replace(/<\|start_header_id\|>reasoning<\|end_header_id\|>/g, "")
        .replace(/<\|start_header_id\|>assistant<\|end_header_id\|>/g, "")
        .replace(/<\|message\|>/g, "")
        .trim();
}

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
onMounted(async () => {
    if (window.innerWidth < 768) {
        sidebarCollapsed.value = true;
    }

    // 初始化 ACP store
    await acpStore.init();

    // WebSocket 模式初始化
    connectWs();
    checkApiStatus();
    await chatStore.loadSessions();
    if (chatStore.currentSessionId) {
        await chatStore.loadMessages(chatStore.currentSessionId);
    }
});
</script>
