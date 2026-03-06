<template>
  <div class="w-full min-h-screen bg-bg-primary text-text-primary flex">
    <!-- 左侧导航 -->
    <aside class="w-64 border-r border-border bg-bg-secondary flex-shrink-0">
      <div class="p-4 border-b border-border">
        <div class="flex items-center gap-2">
          <button
            @click="router.back()"
            class="p-1.5 rounded-lg hover:bg-bg-tertiary transition-colors"
          >
            <ArrowLeftIcon :size="18" />
          </button>
          <h1 class="text-lg font-semibold">设置</h1>
        </div>
      </div>

      <nav class="p-2">
        <button
          v-for="item in menuItems"
          :key="item.key"
          @click="activeSection = item.key"
          :class="[
            'w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors',
            activeSection === item.key
              ? 'bg-accent/10 text-accent'
              : 'text-text-secondary hover:bg-bg-tertiary hover:text-text-primary',
          ]"
        >
          <component :is="item.icon" :size="18" />
          <span class="text-sm font-medium">{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <!-- 右侧内容 -->
    <main class="flex-1 overflow-y-auto">
      <div class="max-w-3xl mx-auto px-6 py-8">
        <!-- 连接设置 -->
        <section v-if="activeSection === 'connection'" class="space-y-6">
          <div>
            <h2 class="text-xl font-semibold mb-1">连接设置</h2>
            <p class="text-text-secondary text-sm">
              配置 API 和 WebSocket 连接地址
            </p>
          </div>

          <div
            class="bg-bg-secondary rounded-xl border border-border p-6 space-y-4"
          >
            <div>
              <label class="block text-sm text-text-secondary mb-2">
                WebSocket 地址
              </label>
              <input
                v-model="wsUrl"
                type="text"
                placeholder="ws://localhost:8080/ws"
                class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
              />
            </div>

            <div>
              <label class="block text-sm text-text-secondary mb-2">
                API 基础地址
              </label>
              <input
                v-model="apiBase"
                type="text"
                placeholder="http://localhost:8080"
                class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
              />
            </div>

            <div>
              <label class="block text-sm text-text-secondary mb-2">
                用户 ID
              </label>
              <input
                v-model="userId"
                type="text"
                placeholder="user-1"
                class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
              />
            </div>
          </div>

          <!-- 连接状态 -->
          <div class="bg-bg-secondary rounded-xl border border-border p-6">
            <h3 class="text-sm font-medium mb-4">连接状态</h3>
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-text-secondary text-sm">API 状态</span>
                <span
                  :class="[
                    'text-sm',
                    apiHealth === 'ok' ? 'text-green-500' : 'text-red-500',
                  ]"
                >
                  {{ apiHealth === "ok" ? "已连接" : "未连接" }}
                </span>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-text-secondary text-sm">WebSocket</span>
                <span class="text-text-secondary text-sm">{{ wsStatus }}</span>
              </div>
            </div>
          </div>
        </section>

        <!-- 工作区设置 -->
        <section v-if="activeSection === 'workspace'" class="space-y-6">
          <div>
            <h2 class="text-xl font-semibold mb-1">工作区设置</h2>
            <p class="text-text-secondary text-sm">配置工作目录路径</p>
          </div>

          <div
            class="bg-bg-secondary rounded-xl border border-border p-6 space-y-4"
          >
            <div>
              <label class="block text-sm text-text-secondary mb-2">
                工作区路径
              </label>
              <div class="flex gap-2">
                <input
                  v-model="workspace"
                  type="text"
                  placeholder="./workspace"
                  class="flex-1 px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
                />
                <button
                  @click="handleSetWorkspace"
                  :disabled="savingWorkspace"
                  class="px-4 py-2.5 bg-accent hover:bg-accent-hover disabled:opacity-50 rounded-lg text-sm font-medium transition-colors"
                >
                  {{ savingWorkspace ? "保存中..." : "保存" }}
                </button>
              </div>
              <p class="text-xs text-text-secondary mt-2">
                修改工作区后需要重启服务才能生效
              </p>
            </div>
          </div>

          <!-- 配置文件 -->
          <div class="bg-bg-secondary rounded-xl border border-border p-6">
            <div class="flex items-center justify-between mb-4">
              <h3 class="text-sm font-medium">配置文件</h3>
              <button
                @click="loadConfigFile"
                class="text-xs text-accent hover:text-accent-hover transition-colors"
              >
                刷新
              </button>
            </div>

            <div class="mb-3">
              <label class="block text-xs text-text-secondary mb-1"
                >文件路径</label
              >
              <div class="text-sm text-text-secondary">
                {{ configPath || "-" }}
              </div>
            </div>

            <div>
              <label class="block text-xs text-text-secondary mb-2"
                >配置内容 (TOML)</label
              >
              <textarea
                v-model="configContent"
                rows="12"
                class="w-full px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm font-mono focus:outline-none focus:border-accent transition-colors resize-none"
                placeholder="加载配置文件..."
              ></textarea>
            </div>

            <div class="flex justify-end gap-2 mt-3">
              <button
                @click="loadConfigFile"
                class="px-3 py-1.5 text-sm border border-border rounded-lg hover:bg-bg-tertiary transition-colors"
              >
                重置
              </button>
              <button
                @click="handleOverwriteConfig"
                :disabled="savingConfig"
                class="px-3 py-1.5 text-sm bg-accent hover:bg-accent-hover disabled:opacity-50 rounded-lg transition-colors"
              >
                {{ savingConfig ? "保存中..." : "保存配置" }}
              </button>
            </div>
          </div>
        </section>

        <!-- Provider 设置 -->
        <section v-if="activeSection === 'provider'" class="space-y-6">
          <div class="flex items-center justify-between">
            <div>
              <h2 class="text-xl font-semibold mb-1">LLM Provider</h2>
              <p class="text-text-secondary text-sm">管理 AI 模型提供商配置</p>
            </div>
            <button
              @click="openAddProvider"
              class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
            >
              <PlusIcon :size="16" />
              添加 Provider
            </button>
          </div>

          <!-- AI Agent 默认模型设置 -->
          <div class="bg-bg-secondary rounded-xl border border-border p-4">
            <div class="flex items-start justify-between mb-3">
              <div>
                <h3 class="text-sm font-medium mb-1">AI Agent 默认模型</h3>
                <p class="text-xs text-text-secondary">设置 AI Agent 默认使用的模型，此设置优先于 Provider 的默认模型</p>
              </div>
              <button
                @click="openAgentModelDialog"
                class="px-3 py-1.5 bg-accent hover:bg-accent-hover rounded-lg text-xs font-medium transition-colors flex items-center gap-1"
              >
                <EditIcon :size="14" />
                设置
              </button>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <span class="text-text-secondary">当前模型：</span>
              <span v-if="agentDefaultModel" class="text-accent font-medium">
                {{ agentDefaultModel }}
              </span>
              <span v-else class="text-text-muted italic">
                未设置（将使用 Provider 的默认模型）
              </span>
            </div>
          </div>

          <div v-if="loading" class="text-text-secondary text-center py-8">
            加载中...
          </div>

          <div v-else-if="providers.length > 0" class="space-y-3">
            <div
              v-for="provider in providers"
              :key="provider.id"
              class="bg-bg-secondary rounded-xl border border-border p-4"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <div
                    :class="[
                      'w-2 h-2 rounded-full',
                      provider.enabled ? 'bg-green-500' : 'bg-text-muted',
                    ]"
                  ></div>
                  <div>
                    <div class="font-medium">{{ provider.name }}</div>
                    <div class="text-xs text-text-secondary mt-0.5">
                      {{ getProviderModel(provider) }}
                    </div>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <span
                    :class="[
                      'text-xs px-2 py-1 rounded',
                      provider.enabled
                        ? 'bg-green-500/20 text-green-400'
                        : 'bg-bg-tertiary text-text-secondary',
                    ]"
                  >
                    {{ provider.enabled ? "已启用" : "未启用" }}
                  </span>
                  <button
                    @click="openEditProvider(provider)"
                    class="p-1.5 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-accent transition-colors"
                    title="编辑"
                  >
                    <EditIcon :size="16" />
                  </button>
                  <button
                    @click="handleDeleteProvider(provider)"
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
            <div class="text-text-secondary text-sm mb-4">
              暂无 Provider 配置
            </div>
            <button
              @click="openAddProvider"
              class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
            >
              添加第一个 Provider
            </button>
          </div>
        </section>

        <!-- 技能管理 -->
        <section v-if="activeSection === 'skill'" class="space-y-6">
          <div class="flex items-center justify-between">
            <div>
              <h2 class="text-xl font-semibold mb-1">技能管理</h2>
              <p class="text-text-secondary text-sm">
                管理自定义技能，扩展 AI 助手能力
              </p>
            </div>
            <button
              @click="router.push('/skills')"
              class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
            >
              管理技能
            </button>
          </div>

          <div class="bg-bg-secondary rounded-xl border border-border p-4">
            <div class="flex items-center justify-between">
              <div>
                <div class="text-sm font-medium">已启用技能</div>
                <div class="text-xs text-text-secondary mt-0.5">
                  {{ skillStore.enabledSkills.length }} /
                  {{ skillStore.skills.length }}
                </div>
              </div>
              <ChevronRightIcon :size="18" class="text-text-secondary" />
            </div>
          </div>
        </section>

        <!-- 外观设置 -->
        <section v-if="activeSection === 'appearance'" class="space-y-6">
          <div>
            <h2 class="text-xl font-semibold mb-1">外观设置</h2>
            <p class="text-text-secondary text-sm">自定义界面外观</p>
          </div>

          <div class="bg-bg-secondary rounded-xl border border-border p-6">
            <div class="flex items-center justify-between">
              <div>
                <div class="font-medium">主题模式</div>
                <div class="text-sm text-text-secondary mt-1">切换明暗主题</div>
              </div>
              <div
                class="flex items-center gap-1 bg-bg-tertiary rounded-lg p-1"
              >
                <button
                  @click="themeStore.setTheme('light')"
                  :class="[
                    'px-3 py-1.5 rounded-md text-sm transition-colors flex items-center gap-1.5',
                    themeStore.theme === 'light'
                      ? 'bg-accent text-white'
                      : 'text-text-secondary hover:text-text-primary',
                  ]"
                >
                  <SunIcon :size="14" />
                  浅色
                </button>
                <button
                  @click="themeStore.setTheme('dark')"
                  :class="[
                    'px-3 py-1.5 rounded-md text-sm transition-colors flex items-center gap-1.5',
                    themeStore.theme === 'dark'
                      ? 'bg-accent text-white'
                      : 'text-text-secondary hover:text-text-primary',
                  ]"
                >
                  <MoonIcon :size="14" />
                  深色
                </button>
              </div>
            </div>
          </div>
        </section>

        <!-- 渠道管理 -->
        <section v-if="activeSection === 'channel'" class="space-y-6">
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
                  <!-- 渠道类型图标 -->
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
                  <!-- 快速启用/禁用开关 -->
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
        </section>

        <!-- 关于 -->
        <section v-if="activeSection === 'about'" class="space-y-6">
          <div>
            <h2 class="text-xl font-semibold mb-1">关于</h2>
            <p class="text-text-secondary text-sm">icooclaw 版本信息</p>
          </div>

          <div class="bg-bg-secondary rounded-xl border border-border p-6">
            <div class="text-center">
              <div
                class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-accent/20 flex items-center justify-center"
              >
                <SparklesIcon :size="32" class="text-accent" />
              </div>
              <h3 class="text-lg font-semibold">icooclaw</h3>
              <p class="text-text-secondary text-sm mt-1">AI 助手平台</p>
              <p class="text-text-muted text-xs mt-2">版本 1.0.0</p>
            </div>
          </div>
        </section>

        <!-- 底部保存按钮 -->
        <div
          v-if="hasChanges"
          class="fixed bottom-0 left-64 right-0 bg-bg-secondary border-t border-border p-4"
        >
          <div class="max-w-3xl mx-auto flex justify-end gap-3">
            <button
              @click="handleReset"
              class="px-4 py-2 rounded-lg border border-border hover:bg-bg-tertiary transition-colors"
            >
              取消
            </button>
            <button
              @click="handleSave"
              class="px-4 py-2 rounded-lg bg-accent hover:bg-accent-hover transition-colors font-medium"
            >
              保存设置
            </button>
          </div>
        </div>
      </div>
    </main>

    <!-- Channel 编辑对话框 -->
    <div
      v-if="showChannelDialog"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="closeChannelDialog"
    >
      <div
        class="bg-bg-secondary rounded-xl border border-border w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto"
      >
        <div
          class="p-4 border-b border-border sticky top-0 bg-bg-secondary z-10"
        >
          <h2 class="text-lg font-medium">
            {{ editingChannel ? "编辑渠道" : "添加渠道" }}
          </h2>
        </div>
        <div class="p-4 space-y-4">
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
              <!-- 配置步骤说明 -->
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
                <!-- Webhook 配置 -->
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

                <!-- Webhook URL 显示与复制 -->
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

                <!-- App ID -->
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

                <!-- App Secret -->
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

                <!-- Verification Token -->
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

                <!-- Encrypt Key -->
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

              <!-- 功能配置 -->
              <div class="border-t border-border pt-4 mt-4">
                <div class="text-sm font-medium mb-3 text-accent">功能配置</div>
                <div class="space-y-3">
                  <!-- 欢迎消息 -->
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

                  <!-- 功能开关 -->
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

              <!-- 权限说明 -->
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
        <div
          class="p-4 border-t border-border flex justify-end gap-3 sticky bottom-0 bg-bg-secondary"
        >
          <button
            @click="closeChannelDialog"
            class="px-4 py-2 rounded-lg border border-border hover:bg-bg-tertiary transition-colors"
          >
            取消
          </button>
          <button
            @click="handleSaveChannel"
            :disabled="!channelForm.name || savingChannel"
            class="px-4 py-2 bg-accent hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-medium transition-colors"
          >
            {{ savingChannel ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>
    </div>

    <!-- Provider 编辑对话框 -->
    <div
      v-if="showProviderDialog"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="closeProviderDialog"
    >
      <div
        class="bg-bg-secondary rounded-xl border border-border w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto"
      >
        <div
          class="p-4 border-b border-border sticky top-0 bg-bg-secondary z-10"
        >
          <h2 class="text-lg font-medium">
            {{ editingProvider ? "编辑 Provider" : "添加 Provider" }}
          </h2>
        </div>
        <div class="p-4 space-y-4">
          <div>
            <label class="block text-sm text-text-secondary mb-2"
              >Provider 名称</label
            >
            <input
              v-model="providerForm.name"
              type="text"
              placeholder="例如: openai, anthropic, deepseek"
              :disabled="!!editingProvider"
              class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors disabled:opacity-50"
            />
          </div>
          <div class="flex items-center gap-3">
            <input
              v-model="providerForm.enabled"
              type="checkbox"
              id="provider-enabled"
              class="w-4 h-4 rounded border-border bg-bg-tertiary text-accent focus:ring-accent"
            />
            <label for="provider-enabled" class="text-sm">
              启用此 Provider
            </label>
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-2"
              >API Key</label
            >
            <input
              v-model="providerForm.apiKey"
              type="password"
              placeholder="sk-..."
              class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
            />
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-2"
              >API Base URL</label
            >
            <input
              v-model="providerForm.apiBase"
              type="text"
              placeholder="https://api.openai.com/v1"
              class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
            />
            <p class="text-xs text-text-secondary mt-1">
              可选，用于自定义 API 端点
            </p>
          </div>
          <div>
            <label class="block text-sm text-text-secondary mb-2"
              >默认模型</label
            >
            <input
              v-model="providerForm.defaultModel"
              type="text"
              placeholder="例如：gpt-4, claude-sonnet-4-20250514"
              class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
            />
            <p class="text-xs text-text-secondary mt-1">
              可选，此 Provider 的默认模型。如果未设置，将使用第一个支持的模型。
            </p>
          </div>

          <!-- 模型列表 -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <label class="text-sm text-text-secondary">支持的模型</label>
              <button
                @click="addModel"
                type="button"
                class="text-xs text-accent hover:text-accent-hover transition-colors flex items-center gap-1"
              >
                <PlusIcon :size="14" />
                添加模型
              </button>
            </div>

            <div v-if="providerForm.models.length > 0" class="space-y-2">
              <div
                v-for="(model, index) in providerForm.models"
                :key="index"
                class="flex items-center gap-2"
              >
                <input
                  v-model="model.name"
                  type="text"
                  placeholder="模型名称 (如 gpt-4)"
                  class="flex-1 px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <input
                  v-model="model.alias"
                  type="text"
                  placeholder="别名 (可选)"
                  class="w-32 px-3 py-2 bg-bg-tertiary border border-border rounded-lg text-sm focus:outline-none focus:border-accent transition-colors"
                />
                <button
                  @click="removeModel(index)"
                  type="button"
                  class="p-2 rounded-lg hover:bg-bg-tertiary text-text-secondary hover:text-red-500 transition-colors"
                >
                  <XIcon :size="16" />
                </button>
              </div>
            </div>

            <div
              v-else
              class="text-text-secondary text-sm py-3 text-center border border-dashed border-border rounded-lg"
            >
              暂无模型，点击上方按钮添加
            </div>
          </div>
        </div>
        <div
          class="p-4 border-t border-border flex justify-end gap-3 sticky bottom-0 bg-bg-secondary"
        >
          <button
            @click="closeProviderDialog"
            class="px-4 py-2 rounded-lg border border-border hover:bg-bg-tertiary transition-colors"
          >
            取消
          </button>
          <button
            @click="handleSaveProvider"
            :disabled="!providerForm.name"
            class="px-4 py-2 bg-accent hover:bg-accent-hover disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-medium transition-colors"
          >
            {{ savingProvider ? "保存中..." : "保存" }}
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- AI Agent 默认模型对话框 -->
  <div
    v-if="showAgentModelDialog"
    class="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center"
  >
    <div
      class="bg-bg-secondary rounded-2xl border border-border w-full max-w-md mx-4 overflow-hidden shadow-2xl"
    >
      <div
        class="p-4 border-b border-border sticky top-0 bg-bg-secondary z-10"
      >
        <h2 class="text-lg font-medium">设置 AI Agent 默认模型</h2>
      </div>
      <div class="p-4 space-y-4">
        <div>
          <label class="block text-sm text-text-secondary mb-2">
            默认模型
          </label>
          <input
            v-model="agentModelForm.model"
            type="text"
            placeholder="例如：gpt-4, claude-sonnet-4-20250514"
            class="w-full px-4 py-2.5 bg-bg-tertiary border border-border rounded-lg focus:outline-none focus:border-accent transition-colors"
          />
          <p class="text-xs text-text-secondary mt-1">
            设置后将优先使用此模型，优先级高于 Provider 的默认模型
          </p>
        </div>
        <div v-if="agentModelForm.providers.length > 0" class="space-y-2">
          <label class="block text-sm text-text-secondary mb-2">
            可选：从 Provider 支持的模型中选择
          </label>
          <div
            v-for="(provider, idx) in agentModelForm.providers"
            :key="idx"
            class="bg-bg-tertiary rounded-lg p-3"
          >
            <div class="text-xs font-medium text-text-secondary mb-2">
              {{ provider.name }}
            </div>
            <div class="flex flex-wrap gap-2">
              <button
                v-for="(llm, llmIdx) in provider.llms"
                :key="llmIdx"
                @click="selectModel(provider.name, llm.model)"
                :class="[
                  'px-3 py-1.5 rounded-lg text-xs transition-colors',
                  agentModelForm.model === llm.model
                    ? 'bg-accent text-white'
                    : 'bg-bg-secondary text-text-secondary hover:bg-bg-hover'
                ]"
              >
                {{ `${provider.name}/${llm.model}` }}
              </button>
            </div>
          </div>
        </div>
      </div>
      <div
        class="p-4 border-t border-border flex justify-end gap-3 sticky bottom-0 bg-bg-secondary"
      >
        <button
          @click="closeAgentModelDialog"
          class="px-4 py-2 rounded-lg border border-border hover:bg-bg-tertiary transition-colors"
        >
          取消
        </button>
        <button
          @click="handleSaveAgentModel"
          class="px-4 py-2 bg-accent hover:bg-accent-hover rounded-lg text-sm font-medium transition-colors"
        >
          {{ savingAgentModel ? "保存中..." : "保存" }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch, reactive } from "vue";
import { useRouter } from "vue-router";
import {
  ArrowLeft as ArrowLeftIcon,
  Plus as PlusIcon,
  Edit as EditIcon,
  Trash as TrashIcon,
  X as XIcon,
  Bot as BotIcon,
  Sparkles as SparklesIcon,
  ChevronRight as ChevronRightIcon,
  Moon as MoonIcon,
  Sun as SunIcon,
  Folder as FolderIcon,
  Palette as PaletteIcon,
  Info as InfoIcon,
  Wifi as ConnectionIcon,
  MessageSquare as ChannelIcon,
  Webhook as WebhookIcon,
  Send as TelegramIcon,
  Radio as FeishuIcon,
  Copy as CopyIcon,
  Check as CheckIcon,
} from "lucide-vue-next";

import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";
import { useThemeStore } from "@/stores/theme";
import { useSkillStore } from "@/stores/skill";
import api from "@/services/api";

const router = useRouter();
const chatStore = useChatStore();
const themeStore = useThemeStore();
const skillStore = useSkillStore();
const { status: wsStatus } = useWebSocket();

// 菜单项
const menuItems = [
  { key: "connection", label: "连接设置", icon: ConnectionIcon },
  { key: "workspace", label: "工作区", icon: FolderIcon },
  { key: "provider", label: "LLM Provider", icon: BotIcon },
  { key: "skill", label: "技能管理", icon: SparklesIcon },
  { key: "channel", label: "渠道管理", icon: ChannelIcon },
  { key: "appearance", label: "外观", icon: PaletteIcon },
  { key: "about", label: "关于", icon: InfoIcon },
];

// 当前选中
const activeSection = ref("connection");

// 表单数据
const wsUrl = ref(chatStore.wsUrl);
const apiBase = ref(chatStore.apiBase);
const userId = ref(chatStore.userId);

// 工作区
const workspace = ref("");
const savingWorkspace = ref(false);

// 配置文件
const configPath = ref("");
const configContent = ref("");
const savingConfig = ref(false);

// Provider 数据
const providers = ref([]);
const loading = ref(true);
const apiHealth = ref("checking");

// AI Agent 默认模型数据
const agentDefaultModel = ref("");
const showAgentModelDialog = ref(false);
const savingAgentModel = ref(false);
const agentModelForm = reactive({
  model: "",
  providers: [],
});

// Provider 对话框
const showProviderDialog = ref(false);
const editingProvider = ref(null);
const savingProvider = ref(false);
const providerForm = reactive({
  name: "",
  enabled: true,
  apiKey: "",
  apiBase: "",
  defaultModel: "",
  models: [], // { name: string, alias: string }
});

// 渠道数据
const channels = ref([]);
const loadingChannels = ref(false);
const showChannelDialog = ref(false);
const editingChannel = ref(null);
const savingChannel = ref(false);
const channelErrors = ref([]);
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
    // 飞书扩展配置
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

// 是否有修改
const hasChanges = computed(() => {
  return (
    wsUrl.value !== chatStore.wsUrl ||
    apiBase.value !== chatStore.apiBase ||
    userId.value !== chatStore.userId
  );
});

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
      // 脱敏显示 webhook URL
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

// 敏感信息脱敏
function maskSensitive(value, showLength = 4) {
  if (!value) return "";
  if (value.length <= showLength * 2) {
    return "••••••••";
  }
  return value.slice(0, showLength) + "••••••••" + value.slice(-showLength);
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

  // 端口验证
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

  // 验证配置
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

// 获取 Provider 模型名称
function getProviderModel(provider) {
  try {
    // 优先显示默认模型
    if (provider.default_model) {
      return provider.default_model + " (默认)";
    }
    // 从 LLMs 字段获取
    if (provider.llms && provider.llms.length > 0) {
      const names = provider.llms.map((l) => l.alias || l.name);
      return names.slice(0, 3).join(", ") + (names.length > 3 ? "..." : "");
    }
    // 兼容旧的 config 字段
    const config = JSON.parse(provider.config || "{}");
    return config.model || "-";
  } catch {
    return "-";
  }
}

// 加载 Provider 列表
async function loadProviders() {
  loading.value = true;
  try {
    const response = await api.getProviders();
    providers.value = response.data || [];
    // 加载默认模型后，更新 providers 到表单
    await loadAgentDefaultModel();
  } catch (error) {
    console.error("获取 Provider 失败:", error);
    providers.value = [];
  }
  loading.value = false;
}

// 加载 AI Agent 默认模型
async function loadAgentDefaultModel() {
  try {
    const response = await api.getDefaultModel();
    if (response.data && response.data.model) {
      agentDefaultModel.value = response.data.model;
    }
  } catch (error) {
    console.error("获取 AI Agent 默认模型失败:", error);
  }
}

// 检查 API 健康状态
async function checkHealth() {
  try {
    await api.checkHealth();
    apiHealth.value = "ok";
  } catch (error) {
    apiHealth.value = "error";
  }
}

// 加载工作区
async function loadWorkspace() {
  try {
    const response = await api.getWorkspace();
    workspace.value = response.data?.workspace || "";
  } catch (error) {
    console.error("获取工作区失败:", error);
  }
}

// 设置工作区
async function handleSetWorkspace() {
  if (!workspace.value) return;

  savingWorkspace.value = true;
  try {
    await api.setWorkspace(workspace.value);
    alert("工作区已设置，需要重启服务才能生效");
  } catch (error) {
    console.error("设置工作区失败:", error);
    alert("设置工作区失败: " + error.message);
  }
  savingWorkspace.value = false;
}

// 加载配置文件
async function loadConfigFile() {
  try {
    const response = await api.getConfigFile();
    configPath.value = response.data?.path || "";
    configContent.value = response.data?.content || "";
  } catch (error) {
    console.error("获取配置文件失败:", error);
    configContent.value = "";
  }
}

// 覆盖配置文件
async function handleOverwriteConfig() {
  if (!configContent.value) return;

  savingConfig.value = true;
  try {
    await api.overwriteConfig(configContent.value);
    alert("配置文件已保存");
  } catch (error) {
    console.error("保存配置文件失败:", error);
    alert("保存配置文件失败: " + error.message);
  }
  savingConfig.value = false;
}

// ===== AI Agent 默认模型相关函数 =====

// 打开 AI Agent 默认模型对话框
function openAgentModelDialog() {
  agentModelForm.model = agentDefaultModel.value || "";
  agentModelForm.providers = providers.value.filter(p => p.enabled && p.llms && p.llms.length > 0);
  showAgentModelDialog.value = true;
}

// 关闭 AI Agent 默认模型对话框
function closeAgentModelDialog() {
  showAgentModelDialog.value = false;
}

// 选择模型
function selectModel(provider, model) {
  agentModelForm.model = `${provider}/${model}`;
}

// 保存 AI Agent 默认模型
async function handleSaveAgentModel() {
  if (!agentModelForm.model) {
    alert("请输入模型名称");
    return;
  }

  savingAgentModel.value = true;
  try {
    await api.setDefaultModel({
      provider_id: null,
      model: agentModelForm.model,
    });
    agentDefaultModel.value = agentModelForm.model;
    closeAgentModelDialog();
    alert("AI Agent 默认模型设置成功");
  } catch (error) {
    console.error("设置 AI Agent 默认模型失败:", error);
    alert("设置失败：" + error.message);
  }
  savingAgentModel.value = false;
}

// 打开添加 Provider 对话框
function openAddProvider() {
  editingProvider.value = null;
  providerForm.name = "";
  providerForm.enabled = true;
  providerForm.apiKey = "";
  providerForm.apiBase = "";
  providerForm.defaultModel = "";
  providerForm.models = [];
  showProviderDialog.value = true;
}

// 打开编辑 Provider 对话框
function openEditProvider(provider) {
  editingProvider.value = provider;
  providerForm.name = provider.name;
  providerForm.enabled = provider.enabled;
  providerForm.apiKey = provider.api_key || "";
  providerForm.apiBase = provider.base_url || "";
  providerForm.defaultModel = provider.default_model || "";

  // 解析 LLMs 字段
  if (provider.llms && provider.llms.length > 0) {
    providerForm.models = provider.llms.map((l) => ({
      name: l.name || "",
      alias: l.model || "", // alias 字段在 storage 中是 model
    }));
  } else {
    providerForm.models = [];
  }

  showProviderDialog.value = true;
}

// 关闭 Provider 对话框
function closeProviderDialog() {
  showProviderDialog.value = false;
  editingProvider.value = null;
}

// 添加模型
function addModel() {
  providerForm.models.push({ name: "", alias: "" });
}

// 删除模型
function removeModel(index) {
  providerForm.models.splice(index, 1);
}

// 保存 Provider
async function handleSaveProvider() {
  if (!providerForm.name) return;

  savingProvider.value = true;

  // 构建 LLMs 数据
  const llms = providerForm.models
    .filter((m) => m.name) // 过滤空模型
    .map((m) => ({
      name: m.name,
      model: m.alias || m.name, // alias 存储到 model 字段
    }));

  const data = {
    name: providerForm.name,
    enabled: providerForm.enabled,
    api_key: providerForm.apiKey,
    base_url: providerForm.apiBase,
    default_model: providerForm.defaultModel,
    llms: llms,
  };

  try {
    if (editingProvider.value) {
      // 更新
      await api.updateProvider({
        id: editingProvider.value.id,
        ...data,
      });
    } else {
      // 创建
      await api.createProvider(data);
    }

    await loadProviders();
    closeProviderDialog();
  } catch (error) {
    console.error("保存 Provider 失败:", error);
    alert("保存 Provider 失败: " + error.message);
  }

  savingProvider.value = false;
}

// 删除 Provider
async function handleDeleteProvider(provider) {
  if (!confirm(`确定要删除 Provider "${provider.name}" 吗？`)) {
    return;
  }

  try {
    await api.deleteProvider(provider.id);
    await loadProviders();
  } catch (error) {
    console.error("删除 Provider 失败:", error);
    alert("删除 Provider 失败: " + error.message);
  }
}

// 重置表单
function handleReset() {
  wsUrl.value = chatStore.wsUrl;
  apiBase.value = chatStore.apiBase;
  userId.value = chatStore.userId;
}

// 保存设置
function handleSave() {
  chatStore.setWsUrl(wsUrl.value);
  chatStore.setApiBase(apiBase.value);
  chatStore.setUserId(userId.value);
  router.push("/");
}

// 监听菜单切换，重新加载对应数据
watch(activeSection, (newVal) => {
  if (newVal === "workspace") {
    loadConfigFile();
  } else if (newVal === "provider") {
    loadProviders();
  } else if (newVal === "skill") {
    skillStore.fetchSkills();
  } else if (newVal === "connection") {
    checkHealth();
  }
});

onMounted(() => {
  loadProviders();
  checkHealth();
  loadWorkspace();
  skillStore.fetchSkills();
});
</script>
