<template>
  <div class="w-full min-h-screen bg-bg-primary text-text-primary">
    <div class="p-6">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-xl font-semibold">定时任务</h1>
          <p class="text-text-secondary text-sm mt-1">管理定时执行的任务</p>
        </div>
        <button
          @click="openAddDialog"
          class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
        >
          <PlusIcon :size="16" />
          新建任务
        </button>
      </div>

      <div v-if="loading" class="text-center py-12 text-text-secondary">
        加载中...
      </div>

      <div v-else-if="tasks.length === 0" class="text-center py-12">
        <Clock :size="48" class="mx-auto text-text-muted mb-4" />
        <p class="text-text-secondary">暂无定时任务</p>
        <p class="text-text-muted text-sm mt-1">点击上方按钮创建第一个任务</p>
      </div>

      <div v-else class="space-y-3">
        <div
          v-for="task in tasks"
          :key="task.id"
          class="bg-bg-secondary rounded-xl border border-border p-4"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div
                :class="[
                  'w-10 h-10 rounded-lg flex items-center justify-center',
                  task.enabled ? 'bg-green-500/10 text-green-500' : 'bg-gray-500/10 text-gray-500'
                ]"
              >
                <Clock :size="20" />
              </div>
              <div>
                <h3 class="font-medium">{{ task.name }}</h3>
                <p class="text-text-secondary text-sm">{{ task.cron_expr }}</p>
                <p v-if="task.description" class="text-text-muted text-xs mt-1">{{ task.description }}</p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <button
                @click="toggleTask(task)"
                :class="[
                  'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
                  task.enabled 
                    ? 'bg-green-500/10 text-green-500 hover:bg-green-500/20' 
                    : 'bg-gray-500/10 text-gray-500 hover:bg-gray-500/20'
                ]"
              >
                {{ task.enabled ? '运行中' : '已暂停' }}
              </button>
              <button
                @click="editTask(task)"
                class="p-1.5 text-text-muted hover:text-text-primary hover:bg-bg-tertiary rounded-lg transition-colors"
              >
                <EditIcon :size="16" />
              </button>
              <button
                @click="deleteTask(task.id)"
                class="p-1.5 text-text-muted hover:text-red-500 hover:bg-red-500/10 rounded-lg transition-colors"
              >
                <TrashIcon :size="16" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建/编辑任务弹窗 -->
    <ModalDialog
      v-model:visible="dialogVisible"
      :title="editingTask ? '编辑任务' : '新建任务'"
      size="md"
      :loading="saving"
      :confirm-disabled="!taskForm.name || !taskForm.cron_expr || !taskForm.channel"
      confirm-text="保存"
      loading-text="保存中..."
      @confirm="saveTask"
    >
      <div class="space-y-4">
        <div>
          <label class="block text-sm text-text-secondary mb-2">任务名称</label>
          <input
            v-model="taskForm.name"
            type="text"
            placeholder="请输入任务名称"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">任务描述</label>
          <input
            v-model="taskForm.description"
            type="text"
            placeholder="请输入任务描述（可选）"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">Cron 表达式</label>
          <input
            v-model="taskForm.cron_expr"
            type="text"
            placeholder="* * * * * (分 时 日 月 周)"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors font-mono"
          />
          <p class="text-xs text-text-muted mt-1">示例: */5 * * * * (每5分钟执行一次)</p>
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">通道</label>
          <input
            v-model="taskForm.channel"
            type="text"
            placeholder="请输入通道名称"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>

        <div>
          <label class="block text-sm text-text-secondary mb-2">参数 (JSON格式)</label>
          <textarea
            v-model="taskForm.params"
            rows="3"
            placeholder='{"key": "value"}'
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors font-mono resize-none"
          ></textarea>
        </div>

        <div class="flex items-center gap-2">
          <input
            v-model="taskForm.enabled"
            type="checkbox"
            id="enabled"
            class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
          />
          <label for="enabled" class="text-sm text-text-secondary">创建后立即启用</label>
        </div>
      </div>
    </ModalDialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from "vue";
import { 
  Clock, 
  Plus as PlusIcon, 
  Edit as EditIcon, 
  Trash as TrashIcon,
} from "lucide-vue-next";
import ModalDialog from "@/components/ModalDialog.vue";
import { 
  getTasks, 
  createTask, 
  updateTask, 
  deleteTask as apiDeleteTask,
  toggleTask as apiToggleTask 
} from "@/services/api.js";

const loading = ref(true);
const tasks = ref([]);
const showAddDialog = ref(false);
const editingTask = ref(null);
const saving = ref(false);

const taskForm = reactive({
  name: "",
  description: "",
  cron_expr: "",
  channel: "",
  params: "",
  enabled: true,
});

// 计算属性：控制弹窗显示
const dialogVisible = computed({
  get: () => showAddDialog.value || !!editingTask.value,
  set: (val) => {
    if (!val) closeDialog();
  }
});

onMounted(() => {
  loadTasks();
});

/**
 * 加载任务列表
 */
async function loadTasks() {
  loading.value = true;
  try {
    const response = await getTasks();
    // 后端返回格式: { code, message, data: [...] }
    const data = response.data || response;
    tasks.value = Array.isArray(data) ? data : [];
  } catch (e) {
    console.error("加载任务失败:", e);
    tasks.value = [];
    alert("加载任务列表失败: " + (e.message || "未知错误"));
  }
  loading.value = false;
}

/**
 * 打开添加任务对话框
 */
function openAddDialog() {
  editingTask.value = null;
  resetForm();
  showAddDialog.value = true;
}

/**
 * 编辑任务
 */
function editTask(task) {
  editingTask.value = task;
  taskForm.name = task.name || "";
  taskForm.description = task.description || "";
  taskForm.cron_expr = task.cron_expr || "";
  taskForm.channel = task.channel || "";
  taskForm.params = task.params || "";
  taskForm.enabled = task.enabled !== false;
  showAddDialog.value = true;
}

/**
 * 重置表单
 */
function resetForm() {
  taskForm.name = "";
  taskForm.description = "";
  taskForm.cron_expr = "";
  taskForm.channel = "";
  taskForm.params = "";
  taskForm.enabled = true;
}

/**
 * 切换任务启用状态
 */
async function toggleTask(task) {
  try {
    await apiToggleTask(task.id);
    // 更新本地状态
    task.enabled = !task.enabled;
  } catch (e) {
    console.error("切换任务状态失败:", e);
    alert("切换任务状态失败: " + (e.message || "未知错误"));
  }
}

/**
 * 删除任务
 */
async function deleteTask(id) {
  if (!confirm("确定要删除这个任务吗？")) {
    return;
  }
  
  try {
    await apiDeleteTask(id);
    // 从本地列表中移除
    tasks.value = tasks.value.filter(t => t.id !== id);
  } catch (e) {
    console.error("删除任务失败:", e);
    alert("删除任务失败: " + (e.message || "未知错误"));
  }
}

/**
 * 关闭对话框
 */
function closeDialog() {
  showAddDialog.value = false;
  editingTask.value = null;
  resetForm();
}

/**
 * 保存任务
 */
async function saveTask() {
  if (!taskForm.name || !taskForm.cron_expr || !taskForm.channel) {
    alert("请填写完整信息（任务名称、Cron表达式、通道为必填项）");
    return;
  }

  // 验证 JSON 参数格式
  if (taskForm.params) {
    try {
      JSON.parse(taskForm.params);
    } catch (e) {
      alert("参数格式错误，请输入有效的 JSON 格式");
      return;
    }
  }

  saving.value = true;

  try {
    const taskData = {
      name: taskForm.name,
      description: taskForm.description,
      cron_expr: taskForm.cron_expr,
      channel: taskForm.channel,
      params: taskForm.params,
      enabled: taskForm.enabled,
    };

    if (editingTask.value) {
      // 更新现有任务
      taskData.id = editingTask.value.id;
      const response = await updateTask(taskData);
      const updatedTask = response.data || taskData;
      
      // 更新本地列表
      const index = tasks.value.findIndex(t => t.id === editingTask.value.id);
      if (index !== -1) {
        tasks.value[index] = { ...tasks.value[index], ...updatedTask };
      }
    } else {
      // 创建新任务
      const response = await createTask(taskData);
      const newTask = response.data || taskData;
      tasks.value.push(newTask);
    }
    
    closeDialog();
  } catch (e) {
    console.error("保存任务失败:", e);
    alert("保存任务失败: " + (e.message || "未知错误"));
  }

  saving.value = false;
}
</script>
