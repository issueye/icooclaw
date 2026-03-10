<template>
    <div class="w-full min-h-screen bg-bg-primary text-text-primary">
        <!-- Header -->
        <header class="border-b border-border bg-bg-secondary">
            <div
                class="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between"
            >
                <div class="flex items-center gap-3">
                    <button
                        @click="router.back()"
                        class="p-2 rounded-lg hover:bg-bg-tertiary transition-colors"
                    >
                        <ArrowLeftIcon :size="20" />
                    </button>
                    <h1 class="text-xl font-semibold">技能管理</h1>
                    <span class="text-sm text-text-secondary">
                        ({{ skillStore.skillCount }})
                    </span>
                </div>
                <div class="flex items-center gap-2">
                    <!-- 导入按钮 -->
                    <button
                        @click="showImportDialog = true"
                        class="px-3 py-2 bg-bg-tertiary hover:bg-bg-tertiary/80 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                    >
                        <UploadIcon :size="16" />
                        导入
                    </button>
                    <!-- 导出按钮 -->
                    <button
                        @click="handleExport"
                        class="px-3 py-2 bg-bg-tertiary hover:bg-bg-tertiary/80 rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                    >
                        <DownloadIcon :size="16" />
                        导出
                    </button>
                    <!-- 添加按钮 -->
                    <button
                        @click="openAddDialog"
                        class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                    >
                        <PlusIcon :size="16" />
                        添加技能
                    </button>
                </div>
            </div>
        </header>

        <main class="max-w-5xl mx-auto px-4 py-6">
            <!-- 工具栏 -->
            <div class="flex flex-col sm:flex-row gap-4 mb-6">
                <!-- 搜索框 -->
                <div class="relative flex-1">
                    <SearchIcon
                        :size="18"
                        class="absolute left-3 top-1/2 -translate-y-1/2 text-text-secondary"
                    />
                    <input
                        v-model="searchKeyword"
                        type="text"
                        placeholder="搜索技能名称、描述或标签..."
                        class="w-full pl-10 pr-4 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                    />
                </div>
                <!-- 筛选器 -->
                <div class="flex gap-2">
                    <select
                        v-model="filterSource"
                        class="px-3 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent"
                    >
                        <option value="">全部来源</option>
                        <option value="builtin">内置</option>
                        <option value="workspace">工作区</option>
                    </select>
                    <select
                        v-model="filterEnabled"
                        class="px-3 py-2.5 bg-bg-secondary border border-border rounded-lg focus:outline-none focus:border-accent"
                    >
                        <option value="">全部状态</option>
                        <option value="true">已启用</option>
                        <option value="false">已禁用</option>
                    </select>
                </div>
            </div>

            <!-- 批量操作栏 -->
            <div
                v-if="selectedSkills.length > 0"
                class="flex items-center justify-between p-3 mb-4 bg-accent/10 border border-accent/20 rounded-lg"
            >
                <span class="text-sm">
                    已选择 <strong>{{ selectedSkills.length }}</strong> 个技能
                </span>
                <div class="flex gap-2">
                    <button
                        @click="batchEnable(true)"
                        class="px-3 py-1.5 text-sm bg-accent hover:bg-accent-hover rounded transition-colors"
                    >
                        启用
                    </button>
                    <button
                        @click="batchEnable(false)"
                        class="px-3 py-1.5 text-sm bg-bg-tertiary hover:bg-bg-tertiary/80 rounded transition-colors"
                    >
                        禁用
                    </button>
                    <button
                        @click="batchAlwaysLoad(true)"
                        class="px-3 py-1.5 text-sm bg-accent hover:bg-accent-hover rounded transition-colors"
                    >
                        始终加载
                    </button>
                    <button
                        @click="batchDelete"
                        class="px-3 py-1.5 text-sm bg-red-500 hover:bg-red-600 rounded transition-colors"
                    >
                        删除
                    </button>
                    <button
                        @click="selectedSkills = []"
                        class="px-3 py-1.5 text-sm text-text-secondary hover:text-text-primary transition-colors"
                    >
                        取消
                    </button>
                </div>
            </div>

            <!-- 标签云 -->
            <div v-if="skillStore.tags.length > 0" class="mb-6">
                <div class="flex flex-wrap gap-2">
                    <button
                        v-for="tag in skillStore.tags"
                        :key="tag"
                        @click="toggleTagFilter(tag)"
                        :class="[
                            'px-3 py-1 text-xs rounded-full transition-colors',
                            selectedTags.includes(tag)
                                ? 'bg-accent text-white'
                                : 'bg-bg-secondary text-text-secondary hover:bg-bg-tertiary'
                        ]"
                    >
                        {{ tag }}
                    </button>
                </div>
            </div>

            <!-- 加载状态 -->
            <div
                v-if="skillStore.loading"
                class="text-center py-8 text-text-secondary"
            >
                <LoaderIcon class="animate-spin mx-auto mb-2" :size="24" />
                加载中...
            </div>

            <!-- 错误提示 -->
            <div
                v-else-if="skillStore.error"
                class="text-center py-8 text-red-500"
            >
                <AlertCircleIcon class="mx-auto mb-2" :size="24" />
                {{ skillStore.error }}
                <button
                    @click="skillStore.fetchSkills()"
                    class="ml-2 text-accent hover:underline"
                >
                    重试
                </button>
            </div>

            <!-- 技能列表 -->
            <div v-else class="space-y-6">
                <!-- 用户技能 -->
                <section v-if="filteredUserSkills.length > 0">
                    <div class="flex items-center justify-between mb-3">
                        <h2 class="text-lg font-medium text-text-secondary">
                            用户技能
                        </h2>
                        <span class="text-sm text-text-secondary">
                            {{ filteredUserSkills.length }}
                        </span>
                    </div>
                    <div class="space-y-3">
                        <SkillCard
                            v-for="skill in filteredUserSkills"
                            :key="skill.id"
                            :skill="skill"
                            :selected="selectedSkills.includes(skill.id)"
                            @toggle-select="toggleSelect(skill.id)"
                            @edit="openEditDialog(skill)"
                            @delete="handleDelete(skill)"
                            @toggle-enabled="toggleSkill(skill)"
                        />
                    </div>
                </section>

                <!-- 内置技能 -->
                <section v-if="filteredBuiltinSkills.length > 0">
                    <div class="flex items-center justify-between mb-3">
                        <h2 class="text-lg font-medium text-text-secondary">
                            内置技能
                        </h2>
                        <span class="text-sm text-text-secondary">
                            {{ filteredBuiltinSkills.length }}
                        </span>
                    </div>
                    <div class="space-y-3">
                        <SkillCard
                            v-for="skill in filteredBuiltinSkills"
                            :key="skill.id"
                            :skill="skill"
                            :selected="selectedSkills.includes(skill.id)"
                            @toggle-select="toggleSelect(skill.id)"
                            @edit="openEditDialog(skill)"
                            @delete="handleDelete(skill)"
                            @toggle-enabled="toggleSkill(skill)"
                        />
                    </div>
                </section>

                <!-- 空状态 -->
                <div
                    v-if="filteredSkills.length === 0"
                    class="text-center py-12"
                >
                    <div
                        class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-bg-tertiary flex items-center justify-center"
                    >
                        <SparklesIcon :size="32" class="text-accent" />
                    </div>
                    <h3 class="text-lg font-medium mb-2">暂无技能</h3>
                    <p class="text-text-secondary text-sm mb-4">
                        {{ searchKeyword ? '没有找到匹配的技能' : '添加自定义技能来扩展 AI 助手的能力' }}
                    </p>
                    <button
                        v-if="!searchKeyword"
                        @click="openAddDialog"
                        class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
                    >
                        添加第一个技能
                    </button>
                </div>
            </div>
        </main>

        <!-- 添加/编辑技能对话框 -->
        <SkillDialog
            v-model:visible="dialogVisible"
            :skill="editingSkill"
            @submit="handleSubmit"
        />

        <!-- 导入对话框 -->
        <ImportDialog
            v-model:visible="showImportDialog"
            @import="handleImport"
        />
    </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from "vue";
import { useRouter } from "vue-router";
import {
    ArrowLeft as ArrowLeftIcon,
    Plus as PlusIcon,
    Sparkles as SparklesIcon,
    Search as SearchIcon,
    Upload as UploadIcon,
    Download as DownloadIcon,
    Loader as LoaderIcon,
    AlertCircle as AlertCircleIcon,
} from "lucide-vue-next";

import { useSkillStore } from "@/stores/skill";
import SkillCard from "@/components/skills/SkillCard.vue";
import SkillDialog from "@/components/skills/SkillDialog.vue";
import ImportDialog from "@/components/skills/ImportDialog.vue";

const router = useRouter();
const skillStore = useSkillStore();

// 搜索和筛选状态
const searchKeyword = ref("");
const filterSource = ref("");
const filterEnabled = ref("");
const selectedTags = ref([]);
const selectedSkills = ref([]);

// 对话框状态
const dialogVisible = ref(false);
const showImportDialog = ref(false);
const editingSkill = ref(null);

// 计算属性：过滤后的技能列表
const filteredSkills = computed(() => {
    let result = skillStore.skills;

    // 搜索过滤
    if (searchKeyword.value) {
        const keyword = searchKeyword.value.toLowerCase();
        result = result.filter(s =>
            s.name?.toLowerCase().includes(keyword) ||
            s.description?.toLowerCase().includes(keyword) ||
            s.tags?.some(t => t.toLowerCase().includes(keyword))
        );
    }

    // 来源过滤
    if (filterSource.value) {
        result = result.filter(s => s.source === filterSource.value);
    }

    // 状态过滤
    if (filterEnabled.value !== "") {
        const enabled = filterEnabled.value === "true";
        result = result.filter(s => s.enabled === enabled);
    }

    // 标签过滤
    if (selectedTags.value.length > 0) {
        result = result.filter(s =>
            selectedTags.value.some(tag => s.tags?.includes(tag))
        );
    }

    return result;
});

const filteredUserSkills = computed(() =>
    filteredSkills.value.filter(s => s.source !== "builtin")
);

const filteredBuiltinSkills = computed(() =>
    filteredSkills.value.filter(s => s.source === "builtin")
);

// 方法
function openAddDialog() {
    editingSkill.value = null;
    dialogVisible.value = true;
}

function openEditDialog(skill) {
    editingSkill.value = skill;
    dialogVisible.value = true;
}

function closeDialog() {
    dialogVisible.value = false;
    editingSkill.value = null;
}

async function handleSubmit(formData) {
    try {
        if (editingSkill.value) {
            await skillStore.updateSkill({
                id: editingSkill.value.id,
                ...formData,
            });
        } else {
            await skillStore.createSkill(formData);
        }
        closeDialog();
    } catch (error) {
        console.error("保存技能失败:", error);
        alert("保存技能失败: " + error.message);
    }
}

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

async function toggleSkill(skill) {
    try {
        await skillStore.toggleSkill(skill.id);
    } catch (error) {
        console.error("更新技能失败:", error);
    }
}

function toggleSelect(id) {
    const index = selectedSkills.value.indexOf(id);
    if (index > -1) {
        selectedSkills.value.splice(index, 1);
    } else {
        selectedSkills.value.push(id);
    }
}

function toggleTagFilter(tag) {
    const index = selectedTags.value.indexOf(tag);
    if (index > -1) {
        selectedTags.value.splice(index, 1);
    } else {
        selectedTags.value.push(tag);
    }
}

// 批量操作
async function batchEnable(enabled) {
    try {
        await skillStore.batchUpdateEnabled(selectedSkills.value, enabled);
        selectedSkills.value = [];
    } catch (error) {
        console.error("批量操作失败:", error);
        alert("批量操作失败: " + error.message);
    }
}

async function batchAlwaysLoad(alwaysLoad) {
    try {
        await skillStore.batchUpdateAlwaysLoad(selectedSkills.value, alwaysLoad);
        selectedSkills.value = [];
    } catch (error) {
        console.error("批量操作失败:", error);
        alert("批量操作失败: " + error.message);
    }
}

async function batchDelete() {
    if (!confirm(`确定要删除选中的 ${selectedSkills.value.length} 个技能吗？`)) {
        return;
    }
    try {
        await skillStore.batchDeleteSkills(selectedSkills.value);
        selectedSkills.value = [];
    } catch (error) {
        console.error("批量删除失败:", error);
        alert("批量删除失败: " + error.message);
    }
}

// 导入导出
async function handleExport() {
    try {
        await skillStore.exportSkills();
    } catch (error) {
        console.error("导出失败:", error);
        alert("导出失败: " + error.message);
    }
}

async function handleImport(file, overwrite) {
    try {
        const result = await skillStore.importSkills(file, overwrite);
        showImportDialog.value = false;
        alert(`导入成功: ${result.success} 个, 跳过: ${result.skip} 个`);
    } catch (error) {
        console.error("导入失败:", error);
        alert("导入失败: " + error.message);
    }
}

// 监听筛选变化，清除选择
watch([searchKeyword, filterSource, filterEnabled, selectedTags], () => {
    selectedSkills.value = [];
});

onMounted(() => {
    skillStore.fetchSkills();
    skillStore.fetchTags();
});
</script>
