<template>
    <div
        class="flex items-center justify-between px-4 py-3 border-b border-border bg-bg-primary flex-shrink-0"
    >
        <div class="flex items-center gap-3">
            <!-- 展开侧边栏按钮（侧边栏收起时显示） -->
            <button
                v-if="sidebarCollapsed"
                @click="$emit('toggle-sidebar')"
                class="text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
            >
                <PanelLeftOpenIcon :size="16" />
            </button>

            <!-- Logo（侧边栏收起时显示） -->
            <div v-if="sidebarCollapsed" class="flex items-center gap-2">
                <div
                    class="w-6 h-6 rounded-lg bg-gradient-to-br from-accent to-[#5b4fcf] flex items-center justify-center flex-shrink-0"
                >
                    <BotIcon :size="12" class="text-white" />
                </div>
            </div>

            <!-- 当前会话标题 -->
            <h1
                class="text-sm font-medium text-text-primary truncate max-w-[300px]"
            >
                {{ title || "新对话" }}
            </h1>
        </div>

        <!-- 右侧操作 -->
        <div class="flex items-center gap-2">
            <!-- WebSocket 连接状态按钮 -->
            <button
                v-if="wsStatus === 'connected'"
                @click="$emit('disconnect')"
                class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-green-600 hover:bg-green-700 rounded-lg transition-colors"
                title="断开连接"
            >
                <WifiIcon :size="14" />
                <span>已连接</span>
            </button>
            <button
                v-else-if="wsStatus === 'connecting' || wsStatus === 'reconnecting'"
                class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-yellow-600 hover:bg-yellow-700 rounded-lg transition-colors cursor-wait"
                title="连接中..."
            >
                <Loader2Icon :size="14" class="animate-spin" />
                <span>连接中</span>
            </button>
            <button
                v-else
                @click="$emit('connect')"
                class="flex items-center gap-1.5 px-2.5 py-1.5 text-xs bg-gray-600 hover:bg-gray-700 rounded-lg transition-colors"
                title="连接"
            >
                <WifiOffIcon :size="14" />
                <span>未连接</span>
            </button>

            <!-- 插槽：模式切换等自定义内容 -->
            <slot name="actions"></slot>

            <!-- 新建对话 -->
            <button
                @click="$emit('new-chat')"
                class="text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
                title="新建对话"
            >
                <SquarePenIcon :size="16" />
            </button>

            <!-- 主题切换 -->
            <button
                @click="themeStore.toggleTheme()"
                class="text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
                title="切换主题"
            >
                <SunIcon v-if="themeStore.theme === 'dark'" :size="16" />
                <MoonIcon v-else :size="16" />
            </button>

            <!-- 设置 -->
            <button
                @click="$emit('open-settings')"
                class="text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
                title="设置"
            >
                <Settings2Icon :size="16" />
            </button>
        </div>
    </div>
</template>

<script setup>
import {
    BotIcon,
    PanelLeftOpenIcon,
    SquarePenIcon,
    Settings2Icon,
    MoonIcon,
    SunIcon,
    Wifi as WifiIcon,
    WifiOff as WifiOffIcon,
    Loader2 as Loader2Icon,
} from "lucide-vue-next";
import { useThemeStore } from "@/stores/theme";

const themeStore = useThemeStore();

defineProps({
    title: { type: String, default: "" },
    sidebarCollapsed: { type: Boolean, default: false },
    apiStatus: { type: String, default: "" },
    wsStatus: { type: String, default: "" },
});

defineEmits(["toggle-sidebar", "new-chat", "open-settings", "connect", "disconnect"]);
</script>
