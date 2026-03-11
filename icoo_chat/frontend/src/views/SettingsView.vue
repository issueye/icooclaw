<template>
  <div class="w-full min-h-screen bg-bg-primary text-text-primary flex">
    <!-- 左侧导航 -->
    <aside class="w-48 border-r border-border bg-bg-secondary flex-shrink-0">
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
        <ConnectionSettings
          v-if="activeSection === 'connection'"
          @connect="handleConnect"
          @disconnect="handleDisconnect"
          @save="handleSaveConnection"
        />

        <!-- Provider 设置 -->
        <ProviderSettings v-if="activeSection === 'provider'" />

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
        <ChannelSettings v-if="activeSection === 'channel'" />

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
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from "vue";
import { useRouter } from "vue-router";
import {
  ArrowLeft as ArrowLeftIcon,
  Bot as BotIcon,
  Sparkles as SparklesIcon,
  ChevronRight as ChevronRightIcon,
  Moon as MoonIcon,
  Sun as SunIcon,
  Palette as PaletteIcon,
  Info as InfoIcon,
  Wifi as ConnectionIcon,
  MessageSquare as ChannelIcon,
} from "lucide-vue-next";

import { useChatStore } from "@/stores/chat";
import { useWebSocket } from "@/composables/useWebSocket";
import { useThemeStore } from "@/stores/theme";
import { useSkillStore } from "@/stores/skill";

import ConnectionSettings from "@/components/settings/ConnectionSettings.vue";
import ProviderSettings from "@/components/settings/ProviderSettings.vue";
import ChannelSettings from "@/components/settings/ChannelSettings.vue";

const router = useRouter();
const emit = defineEmits(["connect-ws", "disconnect-ws"]);
const chatStore = useChatStore();
const themeStore = useThemeStore();
const skillStore = useSkillStore();
const { status: wsStatus } = useWebSocket();

// 菜单项
const menuItems = [
  { key: "connection", label: "连接设置", icon: ConnectionIcon },
  { key: "provider", label: "LLM 供应商", icon: BotIcon },
  { key: "skill", label: "技能管理", icon: SparklesIcon },
  { key: "channel", label: "渠道管理", icon: ChannelIcon },
  { key: "appearance", label: "外观", icon: PaletteIcon },
  { key: "about", label: "关于", icon: InfoIcon },
];

// 当前选中
const activeSection = ref("connection");

// 连接处理
function handleConnect() {
  emit("connect-ws");
}

function handleDisconnect() {
  emit("disconnect-ws");
}

function handleSaveConnection() {
  router.push("/");
}

// 监听菜单切换，加载对应数据
watch(activeSection, (newVal) => {
  if (newVal === "skill") {
    skillStore.fetchSkills();
  }
});

onMounted(() => {
  skillStore.fetchSkills();
});
</script>