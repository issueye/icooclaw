<template>
    <div class="w-full min-h-screen bg-bg-primary text-text-primary">
        <!-- Header -->
        <header class="border-b border-border bg-bg-secondary">
            <div
                class="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between"
            >
                <div class="flex items-center gap-3">
                    <button
                        @click="router.back()"
                        class="p-2 rounded-lg hover:bg-bg-tertiary transition-colors"
                    >
                        <ArrowLeftIcon :size="20" />
                    </button>
                    <h1 class="text-xl font-semibold">技能管理</h1>
                </div>
                <button
                    @click="showAddDialog = true"
                    class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                >
                    <PlusIcon :size="16" />
                    添加技能
                </button>
            </div>
        </header>

        <main class="max-w-4xl mx-auto px-4 py-6">
            <!-- 加载状态 -->
            <div
                v-if="skillStore.loading"
                class="text-center py-8 text-text-secondary"
            >
                加载中...
            </div>

            <!-- 错误提示 -->
            <div
                v-else-if="skillStore.error"
                class="text-center py-8 text-red-500"
            >
                {{ skillStore.error }}
            </div>

            <!-- 技能列表 -->
            <div v-else class="space-y-6">
                <!-- 用户技能 -->
                <section v-if="skillStore.userSkills.length > 0">
                    <h2 class="text-lg font-medium mb-3 text-text-secondary">
                        用户技能
                    </h2>
                    <div class="space-y-3">
                        <div
                            v-for="skill in skillStore.userSkills"
                            :key="skill.id"
                            class="bg-bg-secondary rounded-xl border border-border p-4"
                        >
                            <div class="flex items-start justify-between">
                                <div class="flex-1">
                                    <div class="flex items-center gap-2">
                                        <h3 class="font-medium">
                                            {{ skill.name }}
                                        </h3>
                                        <span
                                            v-if="skill.always_load"
                                            class="text-xs px-2 py-0.5 rounded bg-accent/20 text-accent"
                                        >
                                            始终加载
                                        </span>
                                    </div>
                                    <p class="text-sm text-text-secondary mt-1">
                                        {{ skill.description || "无描述" }}
                                    </p>
                                </div>
                                <div class="flex items-center gap-2">
                                    <!-- 启用/禁用开关 -->
                                    <button
                                        @click="toggleSkill(skill)"
                                        class="w-10 h-5 rounded-full transition-colors relative"
                                        :class="
                                            skill.enabled
                                                ? 'bg-accent'
                                                : 'bg-bg-tertiary'
                                        "
                                    >
                                        <span
                                            class="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform"
                                            :class="
                                                skill.enabled
                                                    ? 'left-5'
                                                    : 'left-0.5'
                                            "
                                        ></span>
                                    </button>
                                    <!-- 编辑按钮 -->
                                    <button
                                        @click="openEditDialog(skill)"
                                        class="p-2 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-accent transition-colors"
                                    >
                                        <EditIcon :size="16" />
                                    </button>
                                    <!-- 删除按钮 -->
                                    <button
                                        @click="handleDelete(skill)"
                                        class="p-2 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-red-500 transition-colors"
                                    >
                                        <TrashIcon :size="16" />
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </section>

                <!-- 内置技能 -->
                <section v-if="skillStore.builtinSkills.length > 0">
                    <h2 class="text-lg font-medium mb-3 text-text-secondary">
                        内置技能
                    </h2>
                    <div class="space-y-3">
                        <div
                            v-for="skill in skillStore.builtinSkills"
                            :key="skill.id"
                            class="bg-bg-secondary rounded-xl border border-border p-4"
                        >
                            <div class="flex items-start justify-between">
                                <div class="flex-1">
                                    <div class="flex items-center gap-2">
                                        <h3 class="font-medium">
                                            {{ skill.name }}
                                        </h3>
                                        <span
                                            class="text-xs px-2 py-0.5 rounded bg-bg-tertiary text-text-secondary"
                                        >
                                            内置
                                        </span>
                                    </div>
                                    <p class="text-sm text-text-secondary mt-1">
                                        {{ skill.description || "无描述" }}
                                    </p>
                                </div>
                                <div class="flex items-center gap-2">
                                    <button
                                        @click="toggleSkill(skill)"
                                        class="w-10 h-5 rounded-full transition-colors relative"
                                        :class="
                                            skill.enabled
                                                ? 'bg-accent'
                                                : 'bg-bg-tertiary'
                                        "
                                    >
                                        <span
                                            class="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform"
                                            :class="
                                                skill.enabled
                                                    ? 'left-5'
                                                    : 'left-0.5'
                                            "
                                        ></span>
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                </section>

                <!-- 空状态 -->
                <div
                    v-if="skillStore.skills.length === 0"
                    class="text-center py-12"
                >
                    <div
                        class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-bg-tertiary flex items-center justify-center"
                    >
                        <SparklesIcon :size="32" class="text-accent" />
                    </div>
                    <h3 class="text-lg font-medium mb-2">暂无技能</h3>
                    <p class="text-text-secondary text-sm mb-4">
                        添加自定义技能来扩展 AI 助手的能力
                    </p>
                    <button
                        @click="showAddDialog = true"
                        class="px-4 py-2 bg-accent hover:bg-[#6b5ce7] rounded-lg text-sm font-medium transition-colors"
                    >
                        添加第一个技能
                    </button>
                </div>
            </div>
        </main>

        <!-- 添加/编辑技能对话框 -->
        <ModalDialog
            v-model:visible="dialogVisible"
            :title="editingSkill ? '编辑技能' : '添加技能'"
            size="lg"
            :confirm-text="editingSkill ? '保存' : '添加'"
            :confirm-disabled="!formData.name || !formData.content"
            @confirm="handleSubmit"
        >
            <div class="space-y-4">
                <div>
                    <label class="block text-sm text-text-secondary mb-2"
                        >技能名称</label
                    >
                    <input
                        v-model="formData.name"
                        type="text"
                        placeholder="例如: code_review"
                        :disabled="!!editingSkill"
                        class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors disabled:opacity-50"
                    />
                </div>
                <div>
                    <label class="block text-sm text-text-secondary mb-2"
                        >描述</label
                    >
                    <input
                        v-model="formData.description"
                        type="text"
                        placeholder="技能功能描述"
                        class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors"
                    />
                </div>
                <div>
                    <label class="block text-sm text-text-secondary mb-2"
                        >技能内容 (Markdown)</label
                    >
                    <textarea
                        v-model="formData.content"
                        placeholder="## 技能名称&#10;技能描述和指令..."
                        rows="8"
                        class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors font-mono text-sm"
                    ></textarea>
                </div>
                <div class="flex items-center gap-2">
                    <input
                        v-model="formData.always_load"
                        type="checkbox"
                        id="always_load"
                        class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-[#7c6af7]"
                    />
                    <label
                        for="always_load"
                        class="text-sm text-text-secondary"
                    >
                        始终加载此技能
                    </label>
                </div>
            </div>
        </ModalDialog>
    </div>
</template>

<script setup>
import { ref, onMounted, reactive, computed } from "vue";
import { useRouter } from "vue-router";
import {
    ArrowLeft as ArrowLeftIcon,
    Plus as PlusIcon,
    Trash as TrashIcon,
    Edit as EditIcon,
    Sparkles as SparklesIcon,
} from "lucide-vue-next";

import { useSkillStore } from "@/stores/skill";
import ModalDialog from "@/components/ModalDialog.vue";

const router = useRouter();
const skillStore = useSkillStore();

// 对话框状态
const showAddDialog = ref(false);
const editingSkill = ref(null);

// 计算属性：控制弹窗显示
const dialogVisible = computed({
    get: () => showAddDialog.value || !!editingSkill.value,
    set: (val) => {
        if (!val) closeDialog();
    }
});

// 表单数据
const formData = reactive({
    name: "",
    description: "",
    content: "",
    always_load: false,
    enabled: true,
    source: "workspace",
});

// 重置表单
function resetForm() {
    formData.name = "";
    formData.description = "";
    formData.content = "";
    formData.always_load = false;
    formData.enabled = true;
    formData.source = "workspace";
}

// 打开编辑对话框
function openEditDialog(skill) {
    editingSkill.value = skill;
    formData.name = skill.name;
    formData.description = skill.description || "";
    formData.content = skill.content || "";
    formData.always_load = skill.always_load || false;
    formData.enabled = skill.enabled;
    formData.source = skill.source || "workspace";
}

// 关闭对话框
function closeDialog() {
    showAddDialog.value = false;
    editingSkill.value = null;
    resetForm();
}

// 提交表单
async function handleSubmit() {
    try {
        if (editingSkill.value) {
            // 更新
            await skillStore.updateSkill({
                id: editingSkill.value.id,
                ...formData,
            });
        } else {
            // 创建
            await skillStore.createSkill({ ...formData });
        }
        closeDialog();
    } catch (error) {
        console.error("保存技能失败:", error);
        alert("保存技能失败: " + error.message);
    }
}

// 删除技能
async function handleDelete(skill) {
    if (confirm(`确定要删除技能 "${skill.name}" 吗？`)) {
        try {
            await skillStore.deleteSkill(skill.id);
        } catch (error) {
            console.error("删除技能失败:", error);
            alert("删除技能失败: " + error.message);
        }
    }
}

// 切换技能启用状态
async function toggleSkill(skill) {
    try {
        await skillStore.toggleSkill(skill.id);
    } catch (error) {
        console.error("更新技能失败:", error);
    }
}

onMounted(() => {
    skillStore.fetchSkills();
});
</script>
