<template>
    <ModalDialog
        :visible="visible"
        @update:visible="$emit('update:visible', $event)"
        title="导入技能"
        confirm-text="导入"
        :confirm-disabled="!selectedFile"
        @confirm="handleImport"
    >
        <div class="space-y-4">
            <!-- 文件选择 -->
            <div>
                <label class="block text-sm text-text-secondary mb-2">
                    选择导入文件 (JSON)
                </label>
                <div
                    @click="$refs.fileInput.click()"
                    @drop.prevent="handleDrop"
                    @dragover.prevent
                    :class="[
                        'border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors',
                        selectedFile ? 'border-accent bg-accent/5' : 'border-border hover:border-accent/50'
                    ]"
                >
                    <input
                        ref="fileInput"
                        type="file"
                        accept=".json"
                        class="hidden"
                        @change="handleFileSelect"
                    />
                    <UploadIcon
                        :size="32"
                        :class="['mx-auto mb-2', selectedFile ? 'text-accent' : 'text-text-secondary']"
                    />
                    <p v-if="selectedFile" class="text-accent font-medium">
                        {{ selectedFile.name }}
                    </p>
                    <p v-else class="text-text-secondary">
                        点击选择文件或拖拽到此处
                    </p>
                    <p class="text-xs text-text-secondary mt-1">
                        支持 .json 格式的技能导出文件
                    </p>
                </div>
            </div>

            <!-- 导入选项 -->
            <div class="flex items-center gap-2">
                <input
                    v-model="overwrite"
                    type="checkbox"
                    id="overwrite"
                    class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
                />
                <label for="overwrite" class="text-sm text-text-secondary cursor-pointer">
                    覆盖已存在的技能
                </label>
            </div>

            <!-- 说明 -->
            <div class="text-xs text-text-secondary bg-bg-tertiary rounded-lg p-3">
                <p class="font-medium mb-1">导入说明：</p>
                <ul class="list-disc list-inside space-y-1">
                    <li>支持从导出文件导入技能</li>
                    <li>内置技能不会被覆盖</li>
                    <li>同名技能根据"覆盖"选项决定是否更新</li>
                </ul>
            </div>
        </div>
    </ModalDialog>
</template>

<script setup>
import { ref } from "vue";
import { Upload as UploadIcon } from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";

/**
 * ImportDialog 组件
 * 用于导入技能的对话框
 * @component
 */

/**
 * 组件属性定义
 */
defineProps({
    /** 对话框是否可见 */
    visible: {
        type: Boolean,
        default: false
    }
});

/**
 * 组件事件定义
 */
const emit = defineEmits([
    /** 更新可见状态 */
    'update:visible',
    /** 导入文件 */
    'import'
]);

// 状态
const selectedFile = ref(null);
const overwrite = ref(false);

/**
 * 处理文件选择
 * @param {Event} event - 文件选择事件
 */
function handleFileSelect(event) {
    const file = event.target.files?.[0];
    if (file) {
        selectedFile.value = file;
    }
}

/**
 * 处理拖拽放置
 * @param {DragEvent} event - 拖拽事件
 */
function handleDrop(event) {
    const file = event.dataTransfer.files?.[0];
    if (file && file.name.endsWith('.json')) {
        selectedFile.value = file;
    }
}

/**
 * 执行导入
 */
function handleImport() {
    if (!selectedFile.value) return;
    
    emit('import', selectedFile.value, overwrite.value);
    
    // 重置状态
    selectedFile.value = null;
    overwrite.value = false;
}
</script>
