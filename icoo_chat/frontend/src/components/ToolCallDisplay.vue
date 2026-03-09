<template>
    <div class="tool-call-container">
        <div
            v-for="toolCall in toolCalls"
            :key="toolCall.id"
            class="tool-call-item mb-2 bg-bg-tertiary rounded-lg border border-border overflow-hidden"
        >
            <!-- 工具头部 -->
            <button
                @click="toggleExpand(toolCall.id)"
                class="w-full flex items-center justify-between px-3 py-2 text-left hover:bg-bg-hover transition-colors"
            >
                <div class="flex items-center gap-2">
                    <!-- 状态图标 -->
                    <div
                        class="w-5 h-5 rounded-full flex items-center justify-center"
                        :class="getStatusClass(toolCall.status)"
                    >
                        <Loader2Icon
                            v-if="toolCall.status === 'running'"
                            :size="12"
                            class="animate-spin"
                        />
                        <CheckIcon
                            v-else-if="toolCall.status === 'completed'"
                            :size="12"
                        />
                        <AlertCircleIcon
                            v-else-if="toolCall.status === 'error'"
                            :size="12"
                        />
                    </div>
                    <!-- 工具名称 -->
                    <span class="text-sm font-medium text-text-primary">
                        {{ toolCall.toolName }}
                    </span>
                    <!-- 状态文本 -->
                    <span
                        class="text-xs"
                        :class="getStatusTextClass(toolCall.status)"
                    >
                        {{ getStatusText(toolCall.status) }}
                    </span>
                </div>
                <ChevronDownIcon
                    :size="14"
                    class="text-text-secondary transition-transform"
                    :class="expandedIds.has(toolCall.id) ? 'rotate-180' : ''"
                />
            </button>

            <!-- 展开内容 -->
            <div
                v-if="expandedIds.has(toolCall.id)"
                class="border-t border-border"
            >
                <!-- 参数 -->
                <div v-if="toolCall.arguments" class="px-3 py-2 border-b border-border">
                    <div class="text-xs text-text-muted mb-1">参数</div>
                    <pre class="text-xs text-text-secondary bg-bg-primary p-2 rounded overflow-x-auto whitespace-pre-wrap break-all">{{ formatArguments(toolCall.arguments) }}</pre>
                </div>

                <!-- 结果 -->
                <div v-if="toolCall.status !== 'running'" class="px-3 py-2">
                    <div class="text-xs text-text-muted mb-1">
                        {{ toolCall.status === 'error' ? '错误' : '结果' }}
                    </div>
                    <pre
                        v-if="toolCall.content || toolCall.error"
                        class="text-xs p-2 rounded overflow-x-auto max-h-48 overflow-y-auto whitespace-pre-wrap break-all"
                        :class="toolCall.status === 'error' ? 'text-red-400 bg-red-900/20' : 'text-text-secondary bg-bg-primary'"
                    >{{ toolCall.error || toolCall.content }}</pre>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { reactive } from 'vue';
import { Loader2Icon, CheckIcon, AlertCircleIcon, ChevronDownIcon } from 'lucide-vue-next';

const props = defineProps({
    toolCalls: {
        type: Array,
        default: () => [],
    },
});

const expandedIds = reactive(new Set());

function toggleExpand(id) {
    if (expandedIds.has(id)) {
        expandedIds.delete(id);
    } else {
        expandedIds.add(id);
    }
}

function getStatusClass(status) {
    switch (status) {
        case 'running':
            return 'bg-yellow-500/20 text-yellow-500';
        case 'completed':
            return 'bg-green-500/20 text-green-500';
        case 'error':
            return 'bg-red-500/20 text-red-500';
        default:
            return 'bg-gray-500/20 text-gray-500';
    }
}

function getStatusTextClass(status) {
    switch (status) {
        case 'running':
            return 'text-yellow-500';
        case 'completed':
            return 'text-green-500';
        case 'error':
            return 'text-red-500';
        default:
            return 'text-text-muted';
    }
}

function getStatusText(status) {
    switch (status) {
        case 'running':
            return '执行中...';
        case 'completed':
            return '完成';
        case 'error':
            return '失败';
        default:
            return status;
    }
}

function formatArguments(args) {
    if (!args) return '';
    try {
        const parsed = JSON.parse(args);
        return JSON.stringify(parsed, null, 2);
    } catch {
        return args;
    }
}
</script>