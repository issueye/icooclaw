import { defineStore } from "pinia";
import { ref, computed } from "vue";
import api from "../services/api";

function generateId() {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8);
}

export const useChatStore = defineStore("chat", () => {
  const sessions = ref([]);
  const currentSessionId = ref(null);
  const isLoading = ref(false);
  const isLoadingSessions = ref(false);
  const apiBase = ref(api.getApiBaseUrl());
  const wsUrl = ref(
    localStorage.getItem("icooclaw_ws_url") || "ws://localhost:8080/ws",
  );
  const userId = ref(localStorage.getItem("icooclaw_user_id") || "user-1");

  const currentSession = computed(
    () => sessions.value.find((s) => s.id === currentSessionId.value) || null,
  );

  const currentMessages = computed(() => currentSession.value?.messages || []);

  async function loadSessions() {
    isLoadingSessions.value = true;
    try {
      const data = await api.getSessions();
      sessions.value = (data || []).map((s) => ({
        id: String(s.id),
        chatId: s.chat_id,
        userId: s.user_id,
        title: s.metadata ? extractTitle(s.metadata) : "新对话",
        messages: [],
        createdAt: new Date(s.created_at).getTime(),
        updatedAt: new Date(s.updated_at).getTime(),
        raw: s,
      }));
      if (sessions.value.length > 0 && !currentSessionId.value) {
        currentSessionId.value = sessions.value[0].id;
      }
    } catch (error) {
      console.error("加载会话列表失败:", error);
    } finally {
      isLoadingSessions.value = false;
    }
  }

  function extractTitle(metadata) {
    try {
      const meta = JSON.parse(metadata);
      return meta.title || "新对话";
    } catch {
      return "新对话";
    }
  }

  async function loadMessages(sessionId) {
    const session = sessions.value.find((s) => s.id === sessionId);
    if (!session) return;

    try {
      const messages = await api.getSessionMessages(sessionId);
      if (messages && Array.isArray(messages)) {
        session.messages = messages.map((m) => ({
          id: String(m.id),
          role: m.role,
          content: m.content,
          thinking: m.reasoning_content || "",
          timestamp: new Date(m.timestamp).getTime(),
          streaming: false,
        }));
      }
    } catch (error) {
      console.error("加载消息失败:", error);
    }
  }

  async function createSession(title = "新对话") {
    try {
      const data = await api.createSession({
        user_id: userId.value,
        metadata: JSON.stringify({ title }),
      });
      const session = {
        id: String(data.id),
        chatId: data.chat_id,
        userId: data.user_id,
        title,
        messages: [],
        createdAt: new Date(data.created_at).getTime(),
        updatedAt: new Date(data.updated_at).getTime(),
        raw: data,
      };
      sessions.value.unshift(session);
      currentSessionId.value = session.id;
      return session;
    } catch (error) {
      console.error("创建会话失败:", error);
      const session = {
        id: generateId(),
        chatId: generateId(),
        userId: userId.value,
        title,
        messages: [],
        createdAt: Date.now(),
        updatedAt: Date.now(),
      };
      sessions.value.unshift(session);
      currentSessionId.value = session.id;
      return session;
    }
  }

  async function switchSession(id) {
    currentSessionId.value = id;
    const session = sessions.value.find((s) => s.id === id);
    if (session && session.messages.length === 0) {
      await loadMessages(id);
    }
  }

  async function deleteSession(id) {
    const idx = sessions.value.findIndex((s) => s.id === id);
    if (idx !== -1) {
      try {
        await api.deleteSession(id);
      } catch (error) {
        console.error("删除会话失败:", error);
      }
      sessions.value.splice(idx, 1);
      if (currentSessionId.value === id) {
        currentSessionId.value = sessions.value[0]?.id || null;
      }
    }
  }

  function updateSessionTitleLocal(id, title) {
    const session = sessions.value.find((s) => s.id === id);
    if (session) {
      session.title = title;
      session.updatedAt = Date.now();
    }
  }

  function addUserMessage(content) {
    if (!currentSession.value) {
      createSession();
      return null;
    }

    const msg = {
      id: generateId(),
      role: "user",
      content,
      timestamp: Date.now(),
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();

    if (currentSession.value.messages.length === 1) {
      const title = content.slice(0, 30) + (content.length > 30 ? "..." : "");
      updateSessionTitleLocal(currentSession.value.id, title);
    }

    return msg;
  }

  function addAIMessage() {
    if (!currentSession.value) return null;

    const msg = {
      id: generateId(),
      role: "assistant",
      content: "",
      thinking: "",
      timestamp: Date.now(),
      streaming: true,
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();
    return msg;
  }

  function appendToLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      lastMsg.content += content;
    }
  }

  function updateThinking(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      lastMsg.thinking = content;
    }
  }

  function finishLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === "assistant") {
      if (content !== undefined) lastMsg.content = content;
      lastMsg.streaming = false;
    }
  }

  function clearCurrentMessages() {
    if (!currentSession.value) return;
    currentSession.value.messages = [];
  }

  function ensureSession() {
    if (!currentSessionId.value || !currentSession.value) {
      createSession();
    }
    return currentSession.value;
  }

  function setApiBase(base) {
    apiBase.value = base;
    api.setApiBaseUrl(base);
    localStorage.setItem("icooclaw_api_base", base);
  }

  function setWsUrl(url) {
    wsUrl.value = url;
    localStorage.setItem("icooclaw_ws_url", url);
  }

  function setUserId(id) {
    userId.value = id;
    localStorage.setItem("icooclaw_user_id", id);
  }

  return {
    sessions,
    currentSessionId,
    isLoading,
    isLoadingSessions,
    apiBase,
    wsUrl,
    userId,
    currentSession,
    currentMessages,
    loadSessions,
    loadMessages,
    createSession,
    switchSession,
    deleteSession,
    updateSessionTitleLocal,
    addUserMessage,
    addAIMessage,
    appendToLastAI,
    updateThinking,
    finishLastAI,
    clearCurrentMessages,
    ensureSession,
    setApiBase,
    setWsUrl,
    setUserId,
  };
});