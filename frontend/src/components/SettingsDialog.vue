<template>
    <!-- 遮罩 -->
    <Transition name="modal">
        <div
            v-if="visible"
            class="fixed inset-0 z-50 flex items-center justify-center"
        >
            <div
                class="absolute inset-0 bg-black/60 backdrop-blur-sm"
                @click="$emit('close')"
            />

            <!-- 弹窗 -->
            <div
                class="relative bg-[#161616] border border-border rounded-2xl shadow-2xl w-full max-w-md mx-4 p-0 overflow-hidden"
            >
                <!-- 标题 -->
                <div
                    class="flex items-center justify-between px-6 py-4 border-b border-border"
                >
                    <div class="flex items-center gap-2.5">
                        <div
                            class="w-7 h-7 rounded-lg bg-accent/15 flex items-center justify-center"
                        >
                            <Settings2Icon :size="14" class="text-accent" />
                        </div>
                        <h2 class="font-semibold text-text-primary">设置</h2>
                    </div>
                    <button
                        @click="$emit('close')"
                        class="text-[#606060] hover:text-text-primary p-1.5 rounded-lg hover:bg-[#2a2a2a] transition-colors"
                    >
                        <XIcon :size="16" />
                    </button>
                </div>

                <div class="px-6 py-5 space-y-5">
                    <!-- WebSocket 服务器地址 -->
                    <div class="space-y-2">
                        <label
                            class="text-sm font-medium text-text-primary flex items-center gap-2"
                        >
                            <WifiIcon :size="14" class="text-accent" />
                            WebSocket 服务器地址
                        </label>
                        <input
                            v-model="localConfig.wsUrl"
                            type="text"
                            placeholder="ws://localhost:8080/ws"
                            class="w-full bg-bg-tertiary border border-border text-text-primary text-sm px-4 py-2.5 rounded-xl outline-none placeholder-[#606060] focus:border-[#7c6af7]/60 transition-colors"
                        />
                        <p class="text-xs text-[#606060]">
                            Agent 后端 WebSocket 端点地址
                        </p>
                    </div>

                    <!-- 用户 ID -->
                    <div class="space-y-2">
                        <label
                            class="text-sm font-medium text-text-primary flex items-center gap-2"
                        >
                            <UserIcon :size="14" class="text-accent" />
                            用户 ID
                        </label>
                        <input
                            v-model="localConfig.userId"
                            type="text"
                            placeholder="user-1"
                            class="w-full bg-bg-tertiary border border-border text-text-primary text-sm px-4 py-2.5 rounded-xl outline-none placeholder-[#606060] focus:border-[#7c6af7]/60 transition-colors"
                        />
                    </div>

                    <!-- 连接状态展示 -->
                    <div
                        class="flex items-center gap-3 bg-[#1a1a1a] border border-border rounded-xl px-4 py-3"
                    >
                        <div
                            class="w-2.5 h-2.5 rounded-full flex-shrink-0"
                            :class="{
                                'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)]':
                                    wsStatus === 'connected',
                                'bg-yellow-500 animate-pulse':
                                    wsStatus === 'connecting',
                                'bg-red-500': wsStatus === 'error',
                                'bg-gray-500': wsStatus === 'disconnected',
                            }"
                        ></div>
                        <div>
                            <div class="text-sm font-medium text-text-primary">
                                {{ statusLabel }}
                            </div>
                            <div
                                v-if="wsError"
                                class="text-xs text-[#ef4444] mt-0.5"
                            >
                                {{ wsError }}
                            </div>
                        </div>
                    </div>
                </div>

                <!-- 底部按钮 -->
                <div class="flex gap-2 px-6 py-4 border-t border-border">
                    <button
                        @click="$emit('close')"
                        class="flex-1 py-2.5 px-4 rounded-xl bg-bg-tertiary border border-border text-sm text-text-secondary hover:text-text-primary hover:bg-[#252525] transition-colors"
                    >
                        取消
                    </button>
                    <button
                        @click="handleSave"
                        class="flex-1 py-2.5 px-4 rounded-xl bg-accent hover:bg-[#6c5ae0] text-sm text-white font-medium transition-colors"
                    >
                        保存并重连
                    </button>
                </div>
            </div>
        </div>
    </Transition>
</template>

<script setup>
import { reactive, computed, watch } from "vue";
import { Settings2Icon, XIcon, WifiIcon, UserIcon } from "lucide-vue-next";

const props = defineProps({
    visible: Boolean,
    wsStatus: { type: String, default: "disconnected" },
    wsError: { type: String, default: null },
    config: {
        type: Object,
        default: () => ({
            wsUrl: "ws://localhost:8080/ws",
            userId: "user-1",
        }),
    },
});

const emit = defineEmits(["close", "save"]);

const localConfig = reactive({ ...props.config });

watch(
    () => props.config,
    (v) => {
        Object.assign(localConfig, v);
    },
    { deep: true },
);

const statusLabel = computed(
    () =>
        ({
            connected: "已连接到 Agent",
            connecting: "正在连接...",
            error: "连接失败",
            disconnected: "未连接",
        })[props.wsStatus] || "未知",
);

function handleSave() {
    emit("save", { ...localConfig });
    emit("close");
}
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
    transition: opacity 0.2s ease;
}
.modal-enter-from,
.modal-leave-to {
    opacity: 0;
}
</style>
