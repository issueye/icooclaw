<template>
    <div
        :class="[
            'bg-bg-secondary rounded-xl border p-4 transition-all',
            selected ? 'border-accent ring-1 ring-accent' : 'border-border hover:border-accent/50'
        ]"
    >
        <div class="flex items-start gap-3">
            <!-- 选择框 -->
            <button
                @click="$emit('toggle-select')"
                :class="[
                    'mt-1 w-5 h-5 rounded border flex items-center justify-center transition-colors',
                    selected
                        ? 'bg-accent border-accent'
                        : 'border-border hover:border-accent'
                ]"
            >
                <CheckIcon v-if="selected" :size="14" class="text-white" />
            </button>

            <!-- 内容区 -->
            <div class="flex-1 min-w-0">
                <div class="flex items-start justify-between gap-2">
                    <div class="flex items-center gap-2 flex-wrap">
                        <h3 class="font-medium">{{ skill.name }}</h3>
                        <!-- 内置标签 -->
                        <span
                            v-if="skill.source === 'builtin'"
                            class="text-xs px-2 py-0.5 rounded bg-bg-tertiary text-text-secondary"
                        >
                            内置
                        </span>
                        <!-- 始终加载标签 -->
                        <span
                            v-if="skill.always_load"
                            class="text-xs px-2 py-0.5 rounded bg-accent/20 text-accent"
                        >
                            始终加载
                        </span>
                    </div>
                    <div class="flex items-center gap-1">
                        <!-- 启用/禁用开关 -->
                        <button
                            @click="$emit('toggle-enabled')"
                            :class="[
                                'w-10 h-5 rounded-full transition-colors relative',
                                skill.enabled ? 'bg-accent' : 'bg-bg-tertiary'
                            ]"
                            :title="skill.enabled ? '已启用' : '已禁用'"
                        >
                            <span
                                class="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform"
                                :class="skill.enabled ? 'left-5' : 'left-0.5'"
                            ></span>
                        </button>
                    </div>
                </div>

                <!-- 描述 -->
                <p class="text-sm text-text-secondary mt-1 line-clamp-2">
                    {{ skill.description || "无描述" }}
                </p>

                <!-- 标签 -->
                <div v-if="skill.tags?.length > 0" class="flex flex-wrap gap-1 mt-2">
                    <span
                        v-for="tag in skill.tags"
                        :key="tag"
                        class="text-xs px-2 py-0.5 rounded bg-bg-tertiary text-text-secondary"
                    >
                        {{ tag }}
                    </span>
                </div>

                <!-- 工具列表 -->
                <div v-if="skill.tools?.length > 0" class="flex flex-wrap gap-1 mt-2">
                    <span class="text-xs text-text-secondary">工具:</span>
                    <span
                        v-for="tool in skill.tools"
                        :key="tool"
                        class="text-xs px-1.5 py-0.5 rounded bg-accent/10 text-accent"
                    >
                        {{ tool }}
                    </span>
                </div>

                <!-- 底部信息 -->
                <div class="flex items-center justify-between mt-3 pt-3 border-t border-border">
                    <div class="flex items-center gap-3 text-xs text-text-secondary">
                        <span v-if="skill.version">v{{ skill.version }}</span>
                        <span v-if="skill.updated_at">
                            更新于 {{ formatDate(skill.updated_at) }}
                        </span>
                    </div>
                    <div class="flex items-center gap-1">
                        <!-- 编辑按钮 -->
                        <button
                            @click="$emit('edit')"
                            class="p-1.5 rounded hover:bg-bg-tertiary text-text-secondary hover:text-accent transition-colors"
                            title="编辑"
                        >
                            <EditIcon :size="14" />
                        </button>
                        <!-- 删除按钮 -->
                        <button
                            v-if="skill.source !== 'builtin'"
                            @click="$emit('delete')"
                            class="p-1.5 rounded hover:bg-bg-tertiary text-text-secondary hover:text-red-500 transition-colors"
                            title="删除"
                        >
                            <TrashIcon :size="14" />
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { Check as CheckIcon, Edit as EditIcon, Trash as TrashIcon } from "lucide-vue-next";

/**
 * SkillCard 组件
 * 用于展示单个技能的信息卡片
 * @component
 */

/**
 * 组件属性定义
 */
const props = defineProps({
    /** 技能数据对象 */
    skill: {
        type: Object,
        required: true
    },
    /** 是否被选中 */
    selected: {
        type: Boolean,
        default: false
    }
});

/**
 * 组件事件定义
 */
defineEmits([
    /** 切换选择状态 */
    'toggle-select',
    /** 编辑技能 */
    'edit',
    /** 删除技能 */
    'delete',
    /** 切换启用状态 */
    'toggle-enabled'
]);

/**
 * 格式化日期
 * @param {string} date - ISO 格式的日期字符串
 * @returns {string} 格式化后的日期字符串
 */
function formatDate(date) {
    if (!date) return '';
    const d = new Date(date);
    const now = new Date();
    const diff = now - d;
    
    // 小于1小时
    if (diff < 3600000) {
        const minutes = Math.floor(diff / 60000);
        return minutes < 1 ? '刚刚' : `${minutes}分钟前`;
    }
    // 小于24小时
    if (diff < 86400000) {
        return `${Math.floor(diff / 3600000)}小时前`;
    }
    // 小于7天
    if (diff < 604800000) {
        return `${Math.floor(diff / 86400000)}天前`;
    }
    
    return d.toLocaleDateString('zh-CN');
}
</script>
