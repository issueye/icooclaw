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
        >
            <ZapIcon :size="14" />
            <span>WebSocket</span>
        </button>
        
        <!-- ACP 模式 -->
        <button
            @click="$emit('update:modelValue', 'acp')"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-all duration-200"
            :class="
                modelValue === 'acp'
                    ? 'bg-accent text-white shadow-sm'
                    : 'text-text-secondary hover:text-text-primary hover:bg-bg-hover'
            "
        >
            <NetworkIcon :size="14" />
            <span>ACP</span>
        </button>
    </div>
</template>

<script setup>
import { ZapIcon, NetworkIcon } from 'lucide-vue-next';

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