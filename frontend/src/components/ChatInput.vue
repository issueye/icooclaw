<template>
    <div
        class="relative flex items-end gap-2 px-4 py-3 border-t border-border bg-bg-primary"
    >
        <!-- 输入框容器 -->
        <div
            class="flex-1 relative flex items-end bg-bg-tertiary border border-border rounded-2xl focus-within:border-[#7c6af7]/60 transition-colors"
        >
            <textarea
                ref="textareaRef"
                v-model="inputText"
                @keydown.enter.exact.prevent="handleSend"
                @keydown.shift.enter="handleNewline"
                @input="autoResize"
                :disabled="disabled"
                placeholder="输入消息... (Enter 发送，Shift+Enter 换行)"
                class="w-full bg-transparent text-text-primary placeholder-[#606060] text-sm px-4 py-3 pr-12 resize-none outline-none leading-relaxed max-h-[200px] min-h-[48px] overflow-y-auto"
                rows="1"
            />

            <!-- 发送按钮 -->
            <button
                @click="handleSend"
                :disabled="!canSend"
                class="absolute right-2 bottom-2 w-8 h-8 rounded-xl flex items-center justify-center transition-all duration-200"
                :class="
                    canSend
                        ? 'bg-accent hover:bg-[#6c5ae0] text-white shadow-lg shadow-[#7c6af7]/20 hover:scale-105'
                        : 'bg-[#2a2a2a] text-[#606060] cursor-not-allowed'
                "
            >
                <LoaderIcon v-if="disabled" :size="14" class="animate-spin" />
                <SendIcon v-else :size="14" />
            </button>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, nextTick } from "vue";
import { SendIcon, LoaderIcon } from "lucide-vue-next";

const props = defineProps({
    disabled: {
        type: Boolean,
        default: false,
    },
});

const emit = defineEmits(["send"]);

const inputText = ref("");
const textareaRef = ref(null);

const canSend = computed(
    () => !props.disabled && inputText.value.trim().length > 0,
);

function handleSend() {
    if (!canSend.value) return;
    const text = inputText.value.trim();
    emit("send", text);
    inputText.value = "";
    nextTick(() => {
        if (textareaRef.value) {
            textareaRef.value.style.height = "auto";
        }
    });
}

function handleNewline() {
    // shift+enter 自然换行，不需要额外处理
}

function autoResize() {
    const el = textareaRef.value;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 200) + "px";
}

// 外部可调用聚焦
function focus() {
    textareaRef.value?.focus();
}

defineExpose({ focus });
</script>
