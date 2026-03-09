import { ref, reactive, computed } from "vue";

const STORAGE_KEY = "icooclaw_chat_sessions";

function loadSessions() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch {}
  return [];
}

function saveSessions(sessions) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(sessions));
  } catch {}
}

function generateId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

// 全局状态（单例）
const sessions = ref(loadSessions());
const currentSessionId = ref(null);
const isLoading = ref(false);

const currentSession = computed(
  () => sessions.value.find((s) => s.id === currentSessionId.value) || null,
);

const currentMessages = computed(() => currentSession.value?.messages || []);

export function useChat() {
  // 创建新会话
  function createSession(title = "新对话") {
    const session = {
      id: generateId(),
      chatId: generateId(), // 发给 agent 的 chat_id
      userId: "user-1",
      title,
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    };
    sessions.value.unshift(session);
    saveSessions(sessions.value);
    currentSessionId.value = session.id;
    return session;
  }

  // 切换会话
  function switchSession(id) {
    currentSessionId.value = id;
  }

  // 删除会话
  function deleteSession(id) {
    const idx = sessions.value.findIndex((s) => s.id === id);
    if (idx !== -1) sessions.value.splice(idx, 1);
    saveSessions(sessions.value);
    if (currentSessionId.value === id) {
      currentSessionId.value = sessions.value[0]?.id || null;
    }
  }

  // 更新会话标题
  function updateSessionTitle(id, title) {
    const session = sessions.value.find((s) => s.id === id);
    if (session) {
      session.title = title;
      session.updatedAt = Date.now();
      saveSessions(sessions.value);
    }
  }

  // 添加用户消息
  function addUserMessage(content) {
    if (!currentSession.value) createSession();
    const msg = {
      id: generateId(),
      role: "user",
      content,
      timestamp: Date.now(),
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();

    // 自动更新标题（第一条消息）
    if (currentSession.value.messages.length === 1) {
      const title = content.slice(0, 30) + (content.length > 30 ? "..." : "");
      updateSessionTitle(currentSession.value.id, title);
    }

    saveSessions(sessions.value);
    return msg;
  }

  // 添加 AI 消息（空消息，后续通过 appendToLastAI 追加内容）
  function addAIMessage() {
    if (!currentSession.value) return null;
    const msg = {
      id: generateId(),
      role: "assistant",
      content: "",
      timestamp: Date.now(),
      streaming: true,
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();
    saveSessions(sessions.value);
    return msg;
  }

  // 追加内容到最后一条 AI 消息
  function appendToLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      lastMsg.content += content;
      saveSessions(sessions.value);
    }
  }

  // 完成最后一条 AI 消息的 streaming
  function finishLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      if (content !== undefined) lastMsg.content = content;
      lastMsg.streaming = false;
      saveSessions(sessions.value);
    }
  }

  // 清空当前会话消息
  function clearCurrentMessages() {
    if (!currentSession.value) return;
    currentSession.value.messages = [];
    saveSessions(sessions.value);
  }

  // 确保当前有一个会话
  function ensureSession() {
    if (!currentSessionId.value || !currentSession.value) {
      createSession();
    }
    return currentSession.value;
  }

  return {
    sessions,
    currentSessionId,
    currentSession,
    currentMessages,
    isLoading,
    createSession,
    switchSession,
    deleteSession,
    updateSessionTitle,
    addUserMessage,
    addAIMessage,
    appendToLastAI,
    finishLastAI,
    clearCurrentMessages,
    ensureSession,
  };
}
