<template>
    <div
        class="flex flex-col h-full bg-bg-primary border-r border-border sidebar-transition"
        :style="{ width: collapsed ? '0' : '260px' }"
    >
        <div
            class="flex flex-col h-full overflow-hidden"
            :class="collapsed ? 'invisible' : 'visible'"
        >
            <!-- 顶部 Logo + 折叠按钮 -->
            <div
                class="flex items-center justify-between px-4 py-4 border-b border-border"
            >
                <div class="flex items-center gap-2.5">
                    <div
                        class="w-7 h-7 rounded-lg bg-gradient-to-br from-accent to-[#5b4fcf] flex items-center justify-center flex-shrink-0"
                    >
                        <BotIcon :size="14" class="text-white" />
                    </div>
                    <span
                        class="font-semibold text-sm text-text-primary tracking-wide"
                        >icooclaw</span
                    >
                </div>
                <button
                    @click="$emit('toggle')"
                    class="text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
                >
                    <PanelLeftCloseIcon :size="16" />
                </button>
            </div>

            <!-- 新建对话按钮 -->
            <div class="px-3 pt-3 pb-2">
                <button
                    @click="handleNewChat"
                    class="w-full flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-sm bg-user-bubble border border-accent/30 text-accent hover:bg-accent/10 hover:border-accent/60 transition-all group"
                >
                    <PlusIcon
                        :size="16"
                        class="group-hover:rotate-90 transition-transform"
                    />
                    新建对话
                </button>
            </div>

            <!-- 会话列表 -->
            <div class="flex-1 overflow-y-auto px-2 pb-2 space-y-0.5">
                <div
                    v-if="sessions.length === 0"
                    class="text-center text-text-muted text-xs py-8"
                >
                    暂无对话记录
                </div>

                <button
                    v-for="session in sessions"
                    :key="session.id"
                    @click="$emit('select', session.id)"
                    class="w-full flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-left group transition-colors text-sm truncate"
                    :class="
                        session.id === currentSessionId
                            ? 'bg-bg-tertiary text-text-primary border border-border-light'
                            : 'text-text-secondary hover:bg-bg-hover hover:text-text-primary'
                    "
                >
                    <MessageSquareIcon
                        :size="14"
                        class="flex-shrink-0 opacity-60"
                    />
                    <span class="flex-1 truncate">{{
                        session.title || "新对话"
                    }}</span>

                    <!-- 删除按钮 -->
                    <span
                        @click.stop="$emit('delete', session.id)"
                        class="opacity-0 group-hover:opacity-100 transition-opacity hover:text-[#ef4444] p-0.5 rounded cursor-pointer"
                    >
                        <Trash2Icon :size="12" />
                    </span>
                </button>
            </div>

            <!-- 底部状态 -->
            <div class="px-3 py-3 border-t border-border">
                <!-- 连接状态 -->
                <div
                    class="flex items-center gap-2 px-3 py-2 rounded-xl bg-bg-secondary"
                >
                    <div
                        class="w-2 h-2 rounded-full flex-shrink-0"
                        :class="{
                            'bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.6)]':
                                wsStatus === 'connected',
                            'bg-yellow-500 animate-pulse':
                                wsStatus === 'connecting',
                            'bg-red-500': wsStatus === 'error',
                            'bg-gray-500': wsStatus === 'disconnected',
                        }"
                    ></div>
                    <span class="text-xs text-text-muted">
                        {{ statusText }}
                    </span>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { computed } from "vue";
import {
    BotIcon,
    PlusIcon,
    PanelLeftCloseIcon,
    MessageSquareIcon,
    Trash2Icon,
} from "lucide-vue-next";

const props = defineProps({
    sessions: { type: Array, default: () => [] },
    currentSessionId: { type: String, default: null },
    wsStatus: { type: String, default: "disconnected" },
    collapsed: { type: Boolean, default: false },
});

const emit = defineEmits(["new", "select", "delete", "toggle"]);

const statusText = computed(
    () =>
        ({
            connected: "Agent 已连接",
            connecting: "连接中...",
            error: "连接失败",
            disconnected: "未连接",
        })[props.wsStatus] || "未知",
);

function handleNewChat() {
    emit("new");
}
</script>
