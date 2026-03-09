<template>
    <div class="mode-switch flex items-center gap-1 bg-bg-tertiary rounded-lg p-1 border border-border">
        <!-- WebSocket 模式 -->
        <button
            @click="$emit('update:modelValue', 'ws')"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-200"
            :class="
                modelValue === 'ws'
                    ? 'bg-accent text-white shadow-sm'
                    : 'text-text-secondary hover:text-text-primary hover:bg-bg-hover'
            "
            title="WebSocket 模式"
        >
            <ZapIcon :size="14" />
            <span>WebSocket</span>
        </button>
    </div>
</template>

<script setup>
import { ZapIcon, PuzzleIcon } from 'lucide-vue-next';

defineProps({
    modelValue: {
        type: String,
        default: 'ws',
    },
});

defineEmits(['update:modelValue']);
</script>

<style scoped>
.mode-switch {
    min-width: 180px;
}
</style>