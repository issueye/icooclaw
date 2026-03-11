<template>
  <section class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-xl font-semibold mb-1">渠道管理</h2>
        <p class="text-text-secondary text-sm">
          管理消息渠道（飞书、Webhook 等）
        </p>
      </div>
      <button
        @click="openAddChannel"
        class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
      >
        <PlusIcon :size="16" />
        添加渠道
      </button>
    </div>

    <div
      v-if="loadingChannels"
      class="text-text-secondary text-center py-8"
    >
      加载中...
    </div>

    <div v-else-if="channels.length > 0" class="space-y-3">
      <div
        v-for="ch in channels"
        :key="ch.id"
        class="bg-bg-secondary rounded-xl border border-border p-4 hover:border-accent/30 transition-colors"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div
              :class="[
                'w-10 h-10 rounded-lg flex items-center justify-center',
                getChannelIconBg(ch),
              ]"
            >
              <component
                :is="getChannelIcon(ch)"
                :size="20"
                :class="ch.enabled ? 'text-accent' : 'text-text-muted'"
              />
            </div>
            <div>
              <div class="font-medium">{{ ch.name }}</div>
              <div class="text-xs text-text-secondary mt-0.5 flex items-center gap-2">
                <span>{{ getChannelTypeLabel(ch) }}</span>
                <span v-if="getChannelEndpoint(ch)" class="text-text-muted">
                  · {{ getChannelEndpoint(ch) }}
                </span>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <button
              @click="toggleChannelEnabled(ch)"
              :class="[
                'relative inline-flex h-5 w-9 items-center rounded-full transition-colors',
                ch.enabled ? 'bg-accent' : 'bg-bg-tertiary',
              ]"
              :title="ch.enabled ? '点击禁用' : '点击启用'"
            >
              <span
                :class="[
                  'inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform',
                  ch.enabled ? 'translate-x-4' : 'translate-x-1',
                ]"
              />
            </button>
            <button
              @click="openEditChannel(ch)"
              class="p-1.5 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-accent transition-colors"
              title="编辑"
            >
              <EditIcon :size="16" />
            </button>
            <button
              @click="handleDeleteChannel(ch)"
              class="p-1.5 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-red-500 transition-colors"
              title="删除"
            >
              <TrashIcon :size="16" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <div
      v-else
      class="bg-bg-secondary rounded-xl border border-border p-8 text-center"
    >
      <div class="text-text-secondary text-sm mb-4">暂无渠道配置</div>
      <button
        @click="openAddChannel"
        class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
      >
        添加第一个渠道
      </button>
    </div>

    <!-- Channel 编辑弹窗 -->
    <ModalDialog
      v-model:visible="channelDialogVisible"
      :title="editingChannel ? '编辑渠道' : '添加渠道'"
      size="lg"
      :scrollable="true"
      :loading="savingChannel"
      :confirm-disabled="!channelForm.name || channelErrors.length > 0"
      confirm-text="保存"
      loading-text="保存中..."
      @confirm="handleSaveChannel"
    >
      <div class="space-y-4">
        <div>
          <label class="block text-sm text-text-secondary mb-2"
            >渠道名称</label
          >
          <input
            v-model="channelForm.name"
            type="text"
            placeholder="如: 飞书客服"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
        </div>
        <div>
          <label class="block text-sm text-text-secondary mb-2"
            >渠道类型</label
          >
          <select
            v-model="channelForm.type"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          >
            <option value="feishu">飞书</option>
            <option value="webhook">Webhook</option>
            <option value="telegram">Telegram</option>
          </select>
        </div>
        <div class="flex items-center gap-3">
          <input
            v-model="channelForm.enabled"
            type="checkbox"
            id="channel-enabled"
            class="w-4 h-4 rounded border-border bg-bg-tertiary"
          />
          <label for="channel-enabled" class="text-sm">启用此渠道</label>
        </div>

        <!-- 飞书专属配置 -->
        <template v-if="channelForm.type === 'feishu'">
          <div class="border-t border-border pt-4">
            <div class="bg-blue-500/10 border border-blue-500/30 rounded-lg p-3 mb-4">
              <div class="text-sm font-medium text-blue-400 mb-2">📋 配置步骤</div>
              <ol class="text-xs text-text-secondary space-y-1 list-decimal list-inside">
                <li>前往 <a href="https://open.feishu.cn/app" target="_blank" class="text-accent hover:underline">飞书开放平台</a> 创建企业自建应用</li>
                <li>在「凭证与基础信息」获取 App ID 和 App Secret</li>
                <li>在「事件订阅」配置请求网址，并获取 Verification Token</li>
                <li>在「权限管理」开通所需权限（im:message, im:message:send_as_bot）</li>
              </ol>
            </div>

            <div class="text-sm font-medium mb-3 text-accent">基础配置</div>
            <div class="space-y-3">
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-xs text-text-secondary mb-1"
                    >监听端口</label
                  >
                  <input
                    v-model.number="channelForm.config.port"
                    type="number"
                    placeholder="8082"
                    class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                  />
                  <p class="text-xs text-text-muted mt-1">本地 Webhook 监听端口</p>
                </div>
                <div>
                  <label class="block text-xs text-text-secondary mb-1"
                    >Webhook 路径</label
                  >
                  <input
                    v-model="channelForm.config.path"
                    type="text"
                    placeholder="/feishu/webhook"
                    class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                  />
                  <p class="text-xs text-text-muted mt-1">事件订阅接收路径</p>
                </div>
              </div>

              <div>
                <label class="block text-xs text-text-secondary mb-1">Webhook 回调地址</label>
                <div class="flex items-center gap-2">
                  <code class="flex-1 bg-bg-tertiary px-3 py-2 rounded-lg text-sm text-text-primary break-all">
                    {{ getWebhookUrl() }}
                  </code>
                  <button
                    type="button"
                    @click="copyWebhookUrl"
                    class="px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm hover:bg-bg-hover transition-colors flex items-center gap-1"
                    :title="webhookUrlCopied ? '已复制' : '复制'"
                  >
                    <component :is="webhookUrlCopied ? CheckIcon : CopyIcon" :size="14" />
                  </button>
                </div>
                <p class="text-xs text-text-muted mt-1">将此地址配置到飞书事件订阅</p>
              </div>

              <div>
                <label class="block text-xs text-text-secondary mb-1">
                  App ID <span class="text-red-400">*</span>
                </label>
                <input
                  v-model="channelForm.config.app_id"
                  type="text"
                  placeholder="cli_xxxxxxxxxxxx"
                  class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <p class="text-xs text-text-muted mt-1">飞书应用凭证，格式：cli_ 开头</p>
              </div>

              <div>
                <label class="block text-xs text-text-secondary mb-1">
                  App Secret <span class="text-red-400">*</span>
                </label>
                <input
                  v-model="channelForm.config.app_secret"
                  type="password"
                  placeholder="应用密钥"
                  class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <p class="text-xs text-text-muted mt-1">在「凭证与基础信息」页面获取</p>
              </div>

              <div>
                <label class="block text-xs text-text-secondary mb-1">
                  Verification Token <span class="text-red-400">*</span>
                </label>
                <input
                  v-model="channelForm.config.verification_token"
                  type="text"
                  placeholder="事件订阅验证 Token"
                  class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <p class="text-xs text-text-muted mt-1">在「事件订阅」页面获取，用于验证请求来源</p>
              </div>

              <div>
                <label class="block text-xs text-text-secondary mb-1">
                  Encrypt Key <span class="text-text-muted">（可选）</span>
                </label>
                <input
                  v-model="channelForm.config.encrypt_key"
                  type="password"
                  placeholder="消息加密密钥"
                  class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <p class="text-xs text-text-muted mt-1">开启消息加密后需要配置，不加密可留空</p>
              </div>
            </div>

            <div class="border-t border-border pt-4 mt-4">
              <div class="text-sm font-medium mb-3 text-accent">功能配置</div>
              <div class="space-y-3">
                <div>
                  <label class="block text-xs text-text-secondary mb-1">
                    欢迎消息 <span class="text-text-muted">（可选）</span>
                  </label>
                  <textarea
                    v-model="channelForm.config.welcome_message"
                    rows="2"
                    placeholder="机器人被添加到群聊时发送的欢迎消息"
                    class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors resize-none"
                  ></textarea>
                </div>

                <div class="space-y-2">
                  <label class="flex items-center gap-3 cursor-pointer">
                    <input
                      v-model="channelForm.config.enable_group_events"
                      type="checkbox"
                      class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
                    />
                    <div>
                      <span class="text-sm">处理群聊事件</span>
                      <p class="text-xs text-text-muted">成员加入/退出、群解散等事件</p>
                    </div>
                  </label>

                  <label class="flex items-center gap-3 cursor-pointer">
                    <input
                      v-model="channelForm.config.enable_card_message"
                      type="checkbox"
                      class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
                    />
                    <div>
                      <span class="text-sm">启用卡片消息</span>
                      <p class="text-xs text-text-muted">支持发送交互式卡片消息</p>
                    </div>
                  </label>
                </div>
              </div>
            </div>

            <div class="border-t border-border pt-4 mt-4">
              <div class="text-sm font-medium mb-3 text-accent">所需权限</div>
              <div class="bg-bg-tertiary rounded-lg p-3">
                <div class="text-xs text-text-secondary space-y-1">
                  <div class="flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-green-500"></span>
                    <code>im:message</code> - 接收消息
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-green-500"></span>
                    <code>im:message:send_as_bot</code> - 以应用身份发消息
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-yellow-500"></span>
                    <code>contact:user.base:readonly</code> - 获取用户信息（可选）
                  </div>
                  <div class="flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-yellow-500"></span>
                    <code>im:chat:readonly</code> - 获取群聊信息（可选）
                  </div>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- Webhook 通用配置 -->
        <template v-else-if="channelForm.type === 'webhook'">
          <div class="border-t border-border pt-4">
            <div class="text-sm font-medium mb-3 text-accent">
              Webhook 配置
            </div>
            <div class="space-y-3">
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-xs text-text-secondary mb-1"
                    >端口</label
                  >
                  <input
                    v-model.number="channelForm.config.port"
                    type="number"
                    placeholder="8081"
                    class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                  />
                </div>
                <div>
                  <label class="block text-xs text-text-secondary mb-1"
                    >路径</label
                  >
                  <input
                    v-model="channelForm.config.path"
                    type="text"
                    placeholder="/webhook"
                    class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                  />
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- Telegram 配置 -->
        <template v-else-if="channelForm.type === 'telegram'">
          <div class="border-t border-border pt-4">
            <div class="text-sm font-medium mb-3 text-accent">
              Telegram 配置
            </div>
            <div class="space-y-3">
              <div>
                <label class="block text-xs text-text-secondary mb-1">
                  Bot Token <span class="text-red-400">*</span>
                </label>
                <input
                  v-model="channelForm.config.bot_token"
                  type="password"
                  placeholder="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
                  class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <p class="text-xs text-text-muted mt-1">
                  从 @BotFather 获取，格式：数字:字母数字组合
                </p>
              </div>
              <div>
                <label class="block text-xs text-text-secondary mb-1">
                  Webhook URL
                </label>
                <input
                  v-model="channelForm.config.webhook_url"
                  type="text"
                  placeholder="https://your-domain.com/api/telegram/webhook"
                  class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <p class="text-xs text-text-muted mt-1">
                  接收 Telegram 消息的回调地址，需要公网可访问
                </p>
              </div>
              <div class="grid grid-cols-2 gap-3">
                <div>
                  <label class="block text-xs text-text-secondary mb-1"
                    >监听端口</label
                  >
                  <input
                    v-model.number="channelForm.config.port"
                    type="number"
                    placeholder="8083"
                    class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                  />
                </div>
                <div>
                  <label class="block text-xs text-text-secondary mb-1"
                    >Webhook 路径</label
                  >
                  <input
                    v-model="channelForm.config.path"
                    type="text"
                    placeholder="/telegram/webhook"
                    class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                  />
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- 配置验证错误提示 -->
        <div
          v-if="channelErrors.length > 0"
          class="bg-red-500/10 border border-red-500/30 rounded-lg p-3"
        >
          <div class="text-sm text-red-400">
            <div v-for="(error, index) in channelErrors" :key="index">
              • {{ error }}
            </div>
          </div>
        </div>
      </div>
    </ModalDialog>
  </section>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from "vue";
import {
  Plus as PlusIcon,
  Edit as EditIcon,
  Trash as TrashIcon,
  Copy as CopyIcon,
  Check as CheckIcon,
  Wifi as ConnectionIcon,
  Webhook as WebhookIcon,
  Send as TelegramIcon,
  Radio as FeishuIcon,
  MessageSquare as ChannelIcon,
} from "lucide-vue-next";
import api from "@/services/api";
import ModalDialog from "@/components/ModalDialog.vue";

// 渠道数据
const channels = ref([]);
const loadingChannels = ref(false);
const showChannelDialog = ref(false);
const editingChannel = ref(null);
const savingChannel = ref(false);
const channelErrors = ref([]);

// Channel 弹窗显示控制
const channelDialogVisible = computed({
  get: () => showChannelDialog.value || !!editingChannel.value,
  set: (val) => {
    if (!val) closeChannelDialog();
  }
});

const channelForm = reactive({
  name: "",
  type: "feishu",
  enabled: true,
  config: {
    port: null,
    path: "",
    app_id: "",
    app_secret: "",
    verification_token: "",
    encrypt_key: "",
    bot_token: "",
    webhook_url: "",
    welcome_message: "",
    enable_group_events: true,
    enable_card_message: true,
  },
});

// Webhook URL 复制状态
const webhookUrlCopied = ref(false);

// 获取 Webhook URL
function getWebhookUrl() {
  const port = channelForm.config.port || 8082;
  const path = channelForm.config.path || "/feishu/webhook";
  return `http://<your-host>:${port}${path}`;
}

// 复制 Webhook URL
async function copyWebhookUrl() {
  const url = getWebhookUrl();
  try {
    await navigator.clipboard.writeText(url);
    webhookUrlCopied.value = true;
    setTimeout(() => {
      webhookUrlCopied.value = false;
    }, 2000);
  } catch (err) {
    console.error("复制失败:", err);
  }
}

// 获取渠道类型显示名
function getChannelTypeLabel(ch) {
  const typeMap = {
    feishu: "飞书",
    webhook: "Webhook",
    telegram: "Telegram",
    websocket: "WebSocket",
  };
  try {
    const cfg =
      typeof ch.config === "string"
        ? JSON.parse(ch.config || "{}")
        : ch.config || {};
    const typeKey = cfg.type || ch.type || "webhook";
    return typeMap[typeKey] || typeKey;
  } catch {
    return ch.type || "Webhook";
  }
}

// 获取渠道图标
function getChannelIcon(ch) {
  const iconMap = {
    feishu: FeishuIcon,
    webhook: WebhookIcon,
    telegram: TelegramIcon,
    websocket: ConnectionIcon,
  };
  try {
    const cfg =
      typeof ch.config === "string"
        ? JSON.parse(ch.config || "{}")
        : ch.config || {};
    const typeKey = cfg.type || ch.type || "webhook";
    return iconMap[typeKey] || ChannelIcon;
  } catch {
    return ChannelIcon;
  }
}

// 获取渠道图标背景色
function getChannelIconBg(ch) {
  const bgMap = {
    feishu: "bg-blue-500/10",
    webhook: "bg-purple-500/10",
    telegram: "bg-sky-500/10",
    websocket: "bg-green-500/10",
  };
  try {
    const cfg =
      typeof ch.config === "string"
        ? JSON.parse(ch.config || "{}")
        : ch.config || {};
    const typeKey = cfg.type || ch.type || "webhook";
    return bgMap[typeKey] || "bg-accent/10";
  } catch {
    return "bg-accent/10";
  }
}

// 获取渠道端点信息
function getChannelEndpoint(ch) {
  try {
    const cfg =
      typeof ch.config === "string"
        ? JSON.parse(ch.config || "{}")
        : ch.config || {};
    if (cfg.port && cfg.path) {
      return `:${cfg.port}${cfg.path}`;
    }
    if (cfg.port) {
      return `:${cfg.port}`;
    }
    if (cfg.webhook_url) {
      return maskUrl(cfg.webhook_url);
    }
    return "";
  } catch {
    return "";
  }
}

// URL 脱敏
function maskUrl(url) {
  if (!url) return "";
  try {
    const urlObj = new URL(url);
    const path = urlObj.pathname;
    if (path.length > 20) {
      return path.slice(0, 15) + "...";
    }
    return path;
  } catch {
    return url.length > 20 ? url.slice(0, 15) + "..." : url;
  }
}

// 验证渠道配置
function validateChannelConfig() {
  channelErrors.value = [];
  const { type, config } = channelForm;

  if (type === "feishu") {
    if (!config.app_id) {
      channelErrors.value.push("App ID 不能为空");
    } else if (!config.app_id.startsWith("cli_")) {
      channelErrors.value.push("App ID 格式不正确，应以 cli_ 开头");
    }
    if (!config.app_secret) {
      channelErrors.value.push("App Secret 不能为空");
    }
    if (!config.verification_token) {
      channelErrors.value.push("Verification Token 不能为空");
    }
  } else if (type === "telegram") {
    if (!config.bot_token) {
      channelErrors.value.push("Bot Token 不能为空");
    } else if (!/^\d+:[A-Za-z0-9_-]+$/.test(config.bot_token)) {
      channelErrors.value.push("Bot Token 格式不正确，应为：数字:字母数字组合");
    }
  }

  if (config.port && (config.port < 1 || config.port > 65535)) {
    channelErrors.value.push("端口号必须在 1-65535 之间");
  }

  return channelErrors.value.length === 0;
}

// 切换渠道启用状态
async function toggleChannelEnabled(ch) {
  const newEnabled = !ch.enabled;
  try {
    await api.updateChannel({
      id: ch.id,
      name: ch.name,
      enabled: newEnabled,
      config: ch.config,
    });
    ch.enabled = newEnabled;
  } catch (error) {
    console.error("切换渠道状态失败:", error);
    alert("操作失败: " + error.message);
  }
}

// 加载渠道列表
async function loadChannels() {
  loadingChannels.value = true;
  try {
    const response = await api.getChannels();
    channels.value = response.data || [];
  } catch (error) {
    console.error("获取渠道失败:", error);
    channels.value = [];
  }
  loadingChannels.value = false;
}

function resetChannelForm() {
  channelForm.name = "";
  channelForm.type = "feishu";
  channelForm.enabled = true;
  channelForm.config.port = null;
  channelForm.config.path = "";
  channelForm.config.app_id = "";
  channelForm.config.app_secret = "";
  channelForm.config.verification_token = "";
  channelForm.config.encrypt_key = "";
  channelForm.config.bot_token = "";
  channelForm.config.webhook_url = "";
  channelForm.config.welcome_message = "";
  channelForm.config.enable_group_events = true;
  channelForm.config.enable_card_message = true;
  channelErrors.value = [];
}

function openAddChannel() {
  editingChannel.value = null;
  resetChannelForm();
  showChannelDialog.value = true;
}

function openEditChannel(ch) {
  editingChannel.value = ch;
  channelForm.name = ch.name;
  channelForm.enabled = ch.enabled;
  channelErrors.value = [];
  try {
    const cfg =
      typeof ch.config === "string"
        ? JSON.parse(ch.config || "{}")
        : ch.config || {};
    channelForm.type = cfg.type || ch.type || "feishu";
    channelForm.config.port = cfg.port || null;
    channelForm.config.path = cfg.path || "";
    channelForm.config.app_id = cfg.app_id || "";
    channelForm.config.app_secret = cfg.app_secret || "";
    channelForm.config.verification_token = cfg.verification_token || "";
    channelForm.config.encrypt_key = cfg.encrypt_key || "";
    channelForm.config.bot_token = cfg.bot_token || "";
    channelForm.config.webhook_url = cfg.webhook_url || "";
    channelForm.config.welcome_message = cfg.welcome_message || "";
    channelForm.config.enable_group_events = cfg.enable_group_events !== false;
    channelForm.config.enable_card_message = cfg.enable_card_message !== false;
  } catch {
    resetChannelForm();
    channelForm.name = ch.name;
    channelForm.enabled = ch.enabled;
  }
  showChannelDialog.value = true;
}

function closeChannelDialog() {
  showChannelDialog.value = false;
  editingChannel.value = null;
  channelErrors.value = [];
}

async function handleSaveChannel() {
  if (!channelForm.name) return;

  if (!validateChannelConfig()) {
    return;
  }

  savingChannel.value = true;
  const data = {
    name: channelForm.name,
    enabled: channelForm.enabled,
    config: JSON.stringify({ type: channelForm.type, ...channelForm.config }),
  };
  try {
    if (editingChannel.value) {
      await api.updateChannel({ id: editingChannel.value.id, ...data });
    } else {
      await api.createChannel(data);
    }
    await loadChannels();
    closeChannelDialog();
  } catch (error) {
    console.error("保存渠道失败:", error);
    alert("保存渠道失败: " + error.message);
  }
  savingChannel.value = false;
}

async function handleDeleteChannel(ch) {
  if (!confirm(`确定要删除渠道 "${ch.name}" 吗？`)) return;
  try {
    await api.deleteChannel(ch.id);
    await loadChannels();
  } catch (error) {
    console.error("删除渠道失败:", error);
    alert("删除渠道失败: " + error.message);
  }
}

onMounted(() => {
  loadChannels();
});
</script>