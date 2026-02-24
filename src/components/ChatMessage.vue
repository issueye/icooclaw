<template>
  <div
    class="message-enter flex gap-3 px-4 py-3"
    :class="isUser ? 'flex-row-reverse' : 'flex-row'"
  >
    <!-- 头像 -->
    <div
      class="flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-semibold"
      :class="
        isUser
          ? 'bg-[#7c6af7] text-white'
          : 'bg-[#1e1e1e] border border-[#2a2a2a] text-[#7c6af7]'
      "
    >
      {{ isUser ? "U" : "AI" }}
    </div>

    <!-- 消息内容 -->
    <div
      class="flex flex-col max-w-[80%]"
      :class="isUser ? 'items-end' : 'items-start'"
    >
      <div
        class="rounded-2xl px-4 py-3 text-sm leading-relaxed relative group"
        :class="
          isUser
            ? 'bg-[#1a1a2e] border border-[#7c6af7]/30 text-[#f0f0f0] rounded-tr-sm'
            : 'bg-[#161616] border border-[#2a2a2a] text-[#f0f0f0] rounded-tl-sm'
        "
      >
        <!-- 用户消息纯文本 -->
        <div v-if="isUser" class="whitespace-pre-wrap">
          {{ message.content }}
        </div>

        <!-- AI 消息 Markdown 渲染 -->
        <div v-else>
          <div
            v-if="message.content"
            class="markdown-content"
            :class="{ 'typing-cursor': message.streaming }"
            v-html="renderedContent"
          />
          <!-- 加载中 dots -->
          <div
            v-if="message.streaming && !message.content"
            class="flex gap-1.5 items-center py-1 px-1"
          >
            <div class="w-2 h-2 rounded-full bg-[#7c6af7] dot-1"></div>
            <div class="w-2 h-2 rounded-full bg-[#7c6af7] dot-2"></div>
            <div class="w-2 h-2 rounded-full bg-[#7c6af7] dot-3"></div>
          </div>
        </div>

        <!-- 复制按钮 -->
        <button
          v-if="!message.streaming && message.content"
          @click="copyContent"
          class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity w-6 h-6 rounded flex items-center justify-center bg-[#2a2a2a] hover:bg-[#333] text-[#909090] hover:text-white"
          :title="copied ? '已复制' : '复制'"
        >
          <CheckIcon v-if="copied" :size="12" />
          <CopyIcon v-else :size="12" />
        </button>
      </div>

      <!-- 时间戳 -->
      <span class="text-xs text-[#606060] mt-1 px-1">{{ timeStr }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from "vue";
import { marked } from "marked";
import hljs from "highlight.js";
import { CopyIcon, CheckIcon } from "lucide-vue-next";

// 配置 marked
marked.setOptions({
  highlight: (code, lang) => {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value;
    }
    return hljs.highlightAuto(code).value;
  },
  breaks: true,
  gfm: true,
});

const props = defineProps({
  message: {
    type: Object,
    required: true,
  },
});

const isUser = computed(() => props.message.role === "user");

const renderedContent = computed(() => {
  if (!props.message.content) return "";
  try {
    return marked.parse(props.message.content);
  } catch {
    return props.message.content;
  }
});

const timeStr = computed(() => {
  const d = new Date(props.message.timestamp);
  return d.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" });
});

const copied = ref(false);
async function copyContent() {
  try {
    await navigator.clipboard.writeText(props.message.content);
    copied.value = true;
    setTimeout(() => (copied.value = false), 2000);
  } catch {}
}
</script>
