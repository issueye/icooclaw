<template>
    <!-- 弹窗组件 - 通用对话框封装 -->
    <Transition name="modal">
        <div
            v-if="visible"
            class="fixed inset-0 z-50 flex items-center justify-center"
            :class="{ 'items-start pt-20': scrollable }"
        >
            <!-- 遮罩层 -->
            <div
                class="absolute inset-0 bg-black/60 backdrop-blur-sm transition-opacity"
                @click="handleMaskClick"
            />

            <!-- 弹窗内容 -->
            <div
                class="relative bg-bg-secondary border border-border rounded-2xl shadow-2xl w-full mx-4 overflow-hidden flex flex-col bg-white"
                :class="[
                    sizeClasses[size],
                    { 'max-h-[90vh]': scrollable }
                ]"
            >
                <!-- 头部 -->
                <div
                    v-if="showHeader"
                    class="flex items-center justify-between px-6 py-4 border-b border-border"
                    :class="{ 'sticky top-0 bg-bg-secondary z-10': scrollable }"
                >
                    <div class="flex items-center gap-2.5">
                        <!-- 图标插槽 -->
                        <slot name="icon">
                            <div
                                v-if="icon"
                                class="w-7 h-7 rounded-lg bg-accent/15 flex items-center justify-center"
                            >
                                <component :is="icon" :size="14" class="text-accent" />
                            </div>
                        </slot>
                        <h2 class="font-semibold text-text-primary">{{ title }}</h2>
                    </div>
                    <!-- 关闭按钮 -->
                    <button
                        v-if="showClose"
                        @click="handleClose"
                        class="text-text-muted hover:text-text-primary p-1.5 rounded-lg hover:bg-bg-tertiary transition-colors"
                    >
                        <XIcon :size="16" />
                    </button>
                </div>

                <!-- 内容区域 -->
                <div
                    class="flex-1"
                    :class="[
                        contentClass,
                        { 'overflow-y-auto': scrollable }
                    ]"
                >
                    <slot />
                </div>

                <!-- 底部按钮 -->
                <div
                    v-if="showFooter"
                    class="flex gap-2 px-6 py-4 border-t border-border"
                    :class="[
                        footerAlignClass,
                        { 'sticky bottom-0 bg-bg-secondary': scrollable }
                    ]"
                >
                    <slot name="footer">
                        <!-- 取消按钮 -->
                        <button
                            v-if="showCancel"
                            @click="handleCancel"
                            class="px-4 py-2 rounded-xl bg-bg-tertiary border border-border text-sm text-text-secondary hover:text-text-primary hover:bg-bg-hover transition-colors"
                            :disabled="loading"
                        >
                            {{ cancelText }}
                        </button>
                        <!-- 确认按钮 -->
                        <button
                            v-if="showConfirm"
                            @click="handleConfirm"
                            :disabled="loading || confirmDisabled"
                            class="px-4 py-2 rounded-xl bg-accent hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed text-sm text-white font-medium transition-colors flex items-center justify-center gap-2"
                        >
                            <Loader2Icon v-if="loading" :size="14" class="animate-spin" />
                            {{ loading ? loadingText : confirmText }}
                        </button>
                    </slot>
                </div>
            </div>
        </div>
    </Transition>
</template>

<script setup>
import { XIcon, Loader2Icon } from "lucide-vue-next";

/**
 * 通用弹窗组件
 * @description 封装了遮罩、头部、内容区、底部按钮的通用对话框
 */

const props = defineProps({
    /** 是否显示弹窗 */
    visible: {
        type: Boolean,
        default: false,
    },
    /** 弹窗标题 */
    title: {
        type: String,
        default: "",
    },
    /** 弹窗尺寸: sm(小), md(中), lg(大), xl(超大), full(全屏) */
    size: {
        type: String,
        default: "md",
        validator: (value) => ["sm", "md", "lg", "xl", "full"].includes(value),
    },
    /** 是否显示头部 */
    showHeader: {
        type: Boolean,
        default: true,
    },
    /** 是否显示关闭按钮 */
    showClose: {
        type: Boolean,
        default: true,
    },
    /** 是否显示底部 */
    showFooter: {
        type: Boolean,
        default: true,
    },
    /** 是否显示取消按钮 */
    showCancel: {
        type: Boolean,
        default: true,
    },
    /** 是否显示确认按钮 */
    showConfirm: {
        type: Boolean,
        default: true,
    },
    /** 取消按钮文本 */
    cancelText: {
        type: String,
        default: "取消",
    },
    /** 确认按钮文本 */
    confirmText: {
        type: String,
        default: "确认",
    },
    /** 加载中文本 */
    loadingText: {
        type: String,
        default: "保存中...",
    },
    /** 是否加载中 */
    loading: {
        type: Boolean,
        default: false,
    },
    /** 确认按钮是否禁用 */
    confirmDisabled: {
        type: Boolean,
        default: false,
    },
    /** 点击遮罩是否关闭 */
    maskClosable: {
        type: Boolean,
        default: true,
    },
    /** 内容区域是否可滚动 */
    scrollable: {
        type: Boolean,
        default: false,
    },
    /** 底部按钮对齐方式: left, center, right */
    footerAlign: {
        type: String,
        default: "right",
        validator: (value) => ["left", "center", "right"].includes(value),
    },
    /** 头部图标组件 */
    icon: {
        type: Object,
        default: null,
    },
    /** 内容区域自定义类名 */
    contentClass: {
        type: String,
        default: "p-6",
    },
});

const emit = defineEmits(["close", "cancel", "confirm", "update:visible"]);

// 尺寸对应的类名
const sizeClasses = {
    sm: "max-w-sm",
    md: "max-w-md",
    lg: "max-w-lg",
    xl: "max-w-xl",
    full: "max-w-4xl",
};

// 底部对齐类名
const footerAlignClass = {
    left: "justify-start",
    center: "justify-center",
    right: "justify-end",
}[props.footerAlign];

/**
 * 处理关闭事件
 */
function handleClose() {
    emit("close");
    emit("update:visible", false);
}

/**
 * 处理遮罩点击
 */
function handleMaskClick() {
    if (props.maskClosable) {
        handleClose();
    }
}

/**
 * 处理取消事件
 */
function handleCancel() {
    emit("cancel");
    handleClose();
}

/**
 * 处理确认事件
 */
function handleConfirm() {
    emit("confirm");
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
