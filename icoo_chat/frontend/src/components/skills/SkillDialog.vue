<template>
    <ModalDialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        :title="skill ? '编辑技能' : '添加技能'"
        size="lg"
        :confirm-text="skill ? '保存' : '添加'"
        :confirm-disabled="!isValid"
        @confirm="handleSubmit"
    >
        <div class="space-y-4">
            <!-- 技能名称 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">
                    技能名称 <span class="text-red-500">*</span>
                </label>
                <input
                    v-model="formData.name"
                    type="text"
                    placeholder="例如: code_review"
                    :disabled="!!skill"
                    class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors disabled:opacity-50"
                />
                <p class="text-xs text-text-secondary mt-1">
                    技能名称只能包含字母、数字、下划线
                </p>
            </div>

            <!-- 描述 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">描述</label>
                <input
                    v-model="formData.description"
                    type="text"
                    placeholder="技能功能描述"
                    class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                />
            </div>

            <!-- 标签 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">标签</label>
                <div class="flex flex-wrap gap-2 mb-2">
                    <span
                        v-for="(tag, index) in formData.tags"
                        :key="index"
                        class="inline-flex items-center gap-1 px-2 py-1 text-sm bg-accent/20 text-accent rounded"
                    >
                        {{ tag }}
                        <button
                            @click="removeTag(index)"
                            class="hover:text-red-500"
                        >
                            <XIcon :size="14" />
                        </button>
                    </span>
                </div>
                <div class="flex gap-2">
                    <input
                        v-model="newTag"
                        type="text"
                        placeholder="添加标签"
                        @keydown.enter.prevent="addTag"
                        class="flex-1 px-4 py-2 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                    />
                    <button
                        @click="addTag"
                        class="px-3 py-2 bg-bg-tertiary hover:bg-bg-tertiary/80 rounded-lg transition-colors"
                    >
                        <PlusIcon :size="16" />
                    </button>
                </div>
            </div>

            <!-- 技能内容 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">
                    技能内容 (Markdown) <span class="text-red-500">*</span>
                </label>
                <textarea
                    v-model="formData.content"
                    placeholder="## 技能名称&#10;技能描述和指令..."
                    rows="8"
                    class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors font-mono text-sm resize-y"
                ></textarea>
            </div>

            <!-- 提示词 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">提示词 (Prompt)</label>
                <textarea
                    v-model="formData.prompt"
                    placeholder="用于 AI 的提示词..."
                    rows="3"
                    class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors font-mono text-sm resize-y"
                ></textarea>
            </div>

            <!-- 工具选择 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">引用的工具</label>
                <div class="flex flex-wrap gap-2 mb-2">
                    <span
                        v-for="(tool, index) in formData.tools"
                        :key="index"
                        class="inline-flex items-center gap-1 px-2 py-1 text-sm bg-accent/20 text-accent rounded"
                    >
                        {{ tool }}
                        <button
                            @click="removeTool(index)"
                            class="hover:text-red-500"
                        >
                            <XIcon :size="14" />
                        </button>
                    </span>
                </div>
                <div class="flex gap-2">
                    <input
                        v-model="newTool"
                        type="text"
                        placeholder="添加工具"
                        @keydown.enter.prevent="addTool"
                        class="flex-1 px-4 py-2 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                    />
                    <button
                        @click="addTool"
                        class="px-3 py-2 bg-bg-tertiary hover:bg-bg-tertiary/80 rounded-lg transition-colors"
                    >
                        <PlusIcon :size="16" />
                    </button>
                </div>
            </div>

            <!-- 复选框选项 -->
            <div class="flex flex-wrap gap-6">
                <label class="flex items-center gap-2 cursor-pointer">
                    <input
                        v-model="formData.always_load"
                        type="checkbox"
                        class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
                    />
                    <span class="text-sm text-text-secondary">始终加载此技能</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                    <input
                        v-model="formData.enabled"
                        type="checkbox"
                        class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
                    />
                    <span class="text-sm text-text-secondary">启用此技能</span>
                </label>
            </div>

            <!-- 版本号 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">版本号</label>
                <input
                    v-model="formData.version"
                    type="text"
                    placeholder="1.0.0"
                    class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                />
            </div>
        </div>
    </ModalDialog>
</template>

<script setup>
import { ref, computed, watch } from "vue";
import { Plus as PlusIcon, X as XIcon } from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";

/**
 * SkillDialog 组件
 * 用于添加或编辑技能的对话框
 * @component
 */

/**
 * 组件属性定义
 */
const props = defineProps({
    /** 对话框是否可见 */
    visible: {
        type: Boolean,
        default: false
    },
    /** 编辑的技能对象，为 null 时表示添加 */
    skill: {
        type: Object,
        default: null
    }
});

/**
 * 组件事件定义
 */
const emit = defineEmits([
    /** 更新可见状态 */
    'update:visible',
    /** 提交表单 */
    'submit'
]);

// 表单数据
const formData = ref({
    name: "",
    description: "",
    content: "",
    prompt: "",
    tags: [],
    tools: [],
    always_load: false,
    enabled: true,
    version: "1.0.0",
    source: "workspace"
});

// 临时输入
const newTag = ref("");
const newTool = ref("");

// 表单验证
const isValid = computed(() => {
    return formData.value.name.trim() && formData.value.content.trim();
});

/**
 * 添加标签
 */
function addTag() {
    const tag = newTag.value.trim();
    if (tag && !formData.value.tags.includes(tag)) {
        formData.value.tags.push(tag);
    }
    newTag.value = "";
}

/**
 * 移除标签
 * @param {number} index - 标签索引
 */
function removeTag(index) {
    formData.value.tags.splice(index, 1);
}

/**
 * 添加工具
 */
function addTool() {
    const tool = newTool.value.trim();
    if (tool && !formData.value.tools.includes(tool)) {
        formData.value.tools.push(tool);
    }
    newTool.value = "";
}

/**
 * 移除工具
 * @param {number} index - 工具索引
 */
function removeTool(index) {
    formData.value.tools.splice(index, 1);
}

/**
 * 提交表单
 */
function handleSubmit() {
    if (!isValid.value) return;
    
    emit("submit", {
        name: formData.value.name.trim(),
        description: formData.value.description.trim(),
        content: formData.value.content.trim(),
        prompt: formData.value.prompt.trim(),
        tags: formData.value.tags,
        tools: formData.value.tools,
        always_load: formData.value.always_load,
        enabled: formData.value.enabled,
        version: formData.value.version,
        source: formData.value.source
    });
}

/**
 * 重置表单
 */
function resetForm() {
    formData.value = {
        name: "",
        description: "",
        content: "",
        prompt: "",
        tags: [],
        tools: [],
        always_load: false,
        enabled: true,
        version: "1.0.0",
        source: "workspace"
    };
    newTag.value = "";
    newTool.value = "";
}

/**
 * 填充表单
 */
function fillForm(skill) {
    if (skill) {
        formData.value = {
            name: skill.name || "",
            description: skill.description || "",
            content: skill.content || "",
            prompt: skill.prompt || "",
            tags: skill.tags || [],
            tools: skill.tools || [],
            always_load: skill.always_load || false,
            enabled: skill.enabled !== false,
            version: skill.version || "1.0.0",
            source: skill.source || "workspace"
        };
    } else {
        resetForm();
    }
}

// 监听 skill 变化
watch(() => props.skill, (newSkill) => {
    fillForm(newSkill);
}, { immediate: true });

// 监听 visible 变化，关闭时重置
watch(() => props.visible, (visible) => {
    if (!visible) {
        resetForm();
    }
});
</script>
