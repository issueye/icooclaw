<template>
  <div class="w-full min-h-screen bg-bg-primary text-text-primary">
    <div class="p-6">
      <div class="flex items-center justify-between mb-6">
        <div>
          <h1 class="text-xl font-semibold">定时任务</h1>
          <p class="text-text-secondary text-sm mt-1">管理定时执行的任务</p>
        </div>
        <button
          @click="showAddDialog = true"
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
                <p class="text-text-secondary text-sm">{{ task.cron }}</p>
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

    <!-- 新建/编辑对话框 -->
    <div v-if="showAddDialog" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="closeDialog">
      <div class="bg-bg-secondary rounded-xl border border-border p-6 w-full max-w-md">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-lg font-semibold">{{ editingTask ? '编辑任务' : '新建任务' }}</h2>
          <button @click="closeDialog" class="text-text-muted hover:text-text-primary">
            <XIcon :size="20" />
          </button>
        </div>

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
            <label class="block text-sm text-text-secondary mb-2">Cron 表达式</label>
            <input
              v-model="taskForm.cron"
              type="text"
              placeholder="* * * * * (分 时 日 月 周)"
              class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors font-mono"
            />
            <p class="text-xs text-text-muted mt-1">示例: */5 * * * * (每5分钟执行一次)</p>
          </div>

          <div>
            <label class="block text-sm text-text-secondary mb-2">执行内容</label>
            <textarea
              v-model="taskForm.content"
              rows="4"
              placeholder="请输入要执行的命令或内容"
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

        <div class="flex justify-end gap-3 mt-6">
          <button
            @click="closeDialog"
            class="px-4 py-2 border border-border rounded-lg hover:bg-bg-tertiary transition-colors"
          >
            取消
          </button>
          <button
            @click="saveTask"
            :disabled="saving"
            class="px-4 py-2 bg-accent hover:bg-accent-hover disabled:opacity-50 rounded-lg font-medium transition-colors"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from "vue";
import { 
  Clock, 
  Plus as PlusIcon, 
  Edit as EditIcon, 
  Trash as TrashIcon,
  X as XIcon 
} from "lucide-vue-next";

const loading = ref(true);
const tasks = ref([]);
const showAddDialog = ref(false);
const editingTask = ref(null);
const saving = ref(false);

const taskForm = reactive({
  name: "",
  cron: "",
  content: "",
  enabled: true,
});

onMounted(() => {
  loadTasks();
});

async function loadTasks() {
  loading.value = true;
  try {
    const savedTasks = localStorage.getItem("icooclaw_tasks");
    if (savedTasks) {
      tasks.value = JSON.parse(savedTasks);
    } else {
      tasks.value = [];
    }
  } catch (e) {
    console.error("加载任务失败:", e);
    tasks.value = [];
  }
  loading.value = false;
}

function saveTasksToStorage() {
  localStorage.setItem("icooclaw_tasks", JSON.stringify(tasks.value));
}

function editTask(task) {
  editingTask.value = task;
  taskForm.name = task.name;
  taskForm.cron = task.cron;
  taskForm.content = task.content;
  taskForm.enabled = task.enabled;
  showAddDialog.value = true;
}

function toggleTask(task) {
  task.enabled = !task.enabled;
  saveTasksToStorage();
}

function deleteTask(id) {
  if (confirm("确定要删除这个任务吗？")) {
    tasks.value = tasks.value.filter(t => t.id !== id);
    saveTasksToStorage();
  }
}

function closeDialog() {
  showAddDialog.value = false;
  editingTask.value = null;
  taskForm.name = "";
  taskForm.cron = "";
  taskForm.content = "";
  taskForm.enabled = true;
}

async function saveTask() {
  if (!taskForm.name || !taskForm.cron || !taskForm.content) {
    alert("请填写完整信息");
    return;
  }

  saving.value = true;

  try {
    if (editingTask.value) {
      const index = tasks.value.findIndex(t => t.id === editingTask.value.id);
      if (index !== -1) {
        tasks.value[index] = {
          ...editingTask.value,
          ...taskForm,
        };
      }
    } else {
      tasks.value.push({
        id: Date.now().toString(),
        ...taskForm,
      });
    }
    saveTasksToStorage();
    closeDialog();
  } catch (e) {
    console.error("保存任务失败:", e);
    alert("保存失败");
  }

  saving.value = false;
}
</script>
