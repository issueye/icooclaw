<template>
    <div
        class="flex items-center justify-between px-4 py-3 border-b border-border bg-bg-primary flex-shrink-0"
    >
        <div class="flex items-center gap-3">
            <!-- 展开侧边栏按钮（侧边栏收起时显示） -->
            <button
                v-if="sidebarCollapsed"
                @click="$emit('toggle-sidebar')"
                class="text-[#606060] hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
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
            <!-- 新建对话 -->
            <button
                @click="$emit('new-chat')"
                class="text-[#606060] hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
                title="新建对话"
            >
                <SquarePenIcon :size="16" />
            </button>

            <!-- 设置 -->
            <button
                @click="$emit('open-settings')"
                class="text-[#606060] hover:text-text-primary hover:bg-bg-tertiary rounded-lg p-1.5 transition-colors"
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
} from "lucide-vue-next";

defineProps({
    title: { type: String, default: "" },
    sidebarCollapsed: { type: Boolean, default: false },
});

defineEmits(["toggle-sidebar", "new-chat", "open-settings"]);
</script>
