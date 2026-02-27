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

                            <!-- 用户技能描述 -->
                            <div
                                v-if="
                                    !skill.builtin &&
                                    skillStore.userSkills.includes(skill)
                                "
                                class="mt-3 pt-3 border-t border-border"
                            >
                                <label class="text-xs text-text-secondary"
                                    >描述</label
                                >
                                <textarea
                                    v-model="skill.description"
                                    @blur="updateSkill(skill)"
                                    rows="2"
                                    class="w-full mt-1 px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm text-text-primary outline-none placeholder-text-muted focus:border-accent/60 resize-none"
                                ></textarea>
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

        <!-- 添加技能对话框 -->
        <div
            v-if="showAddDialog"
            class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
            @click.self="showAddDialog = false"
        >
            <div
                class="bg-bg-secondary rounded-xl border border-border w-full max-w-lg mx-4"
            >
                <div class="p-4 border-b border-border">
                    <h2 class="text-lg font-medium">添加技能</h2>
                </div>
                <div class="p-4 space-y-4">
                    <div>
                        <label class="block text-sm text-text-secondary mb-2"
                            >技能名称</label
                        >
                        <input
                            v-model="newSkill.name"
                            type="text"
                            placeholder="例如: code_review"
                            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors"
                        />
                    </div>
                    <div>
                        <label class="block text-sm text-text-secondary mb-2"
                            >描述</label
                        >
                        <input
                            v-model="newSkill.description"
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
                            v-model="newSkill.content"
                            placeholder="## 技能名称&#10;技能描述和指令..."
                            rows="8"
                            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-[#7c6af7] transition-colors font-mono text-sm"
                        ></textarea>
                    </div>
                    <div class="flex items-center gap-2">
                        <input
                            v-model="newSkill.always_load"
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
                <div class="p-4 border-t border-border flex justify-end gap-3">
                    <button
                        @click="showAddDialog = false"
                        class="px-4 py-2 rounded-lg border border-border hover:bg-bg-tertiary transition-colors"
                    >
                        取消
                    </button>
                    <button
                        @click="handleAddSkill"
                        :disabled="!newSkill.name || !newSkill.content"
                        class="px-4 py-2 bg-accent hover:bg-[#6b5ce7] disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-medium transition-colors"
                    >
                        添加
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import {
    ArrowLeft as ArrowLeftIcon,
    Plus as PlusIcon,
    Trash as TrashIcon,
    Sparkles as SparklesIcon,
} from "lucide-vue-next";

import { useSkillStore } from "@/stores/skill";

const router = useRouter();
const skillStore = useSkillStore();

// 对话框状态
const showAddDialog = ref(false);

// 新技能表单
const newSkill = ref({
    name: "",
    description: "",
    content: "",
    always_load: false,
    enabled: true,
});

// 添加技能
async function handleAddSkill() {
    try {
        await skillStore.createSkill({
            ...newSkill.value,
        });
        showAddDialog.value = false;
        // 重置表单
        newSkill.value = {
            name: "",
            description: "",
            content: "",
            always_load: false,
            enabled: true,
        };
    } catch (error) {
        console.error("添加技能失败:", error);
    }
}

// 删除技能
async function handleDelete(skill) {
    if (confirm(`确定要删除技能 "${skill.name}" 吗？`)) {
        try {
            await skillStore.deleteSkill(skill.id);
        } catch (error) {
            console.error("删除技能失败:", error);
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
