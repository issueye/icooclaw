// 聊天 Store - 使用 Pinia 管理聊天状态

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import api from '../services/api';

const STORAGE_KEY = 'icooclaw_chat_sessions';

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

export const useChatStore = defineStore('chat', () => {
  // ===== 状态 =====
  const sessions = ref(loadSessions());
  const currentSessionId = ref(null);
  const isLoading = ref(false);
  const apiBase = ref(api.getApiBaseUrl());
  const wsUrl = ref(localStorage.getItem('icooclaw_ws_url') || 'ws://localhost:8080/ws');
  const userId = ref(localStorage.getItem('icooclaw_user_id') || 'user-1');

  // ===== 计算属性 =====
  const currentSession = computed(() =>
    sessions.value.find(s => s.id === currentSessionId.value) || null
  );

  const currentMessages = computed(() => currentSession.value?.messages || []);

  // ===== 会话操作 =====
  function createSession(title = '新对话') {
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
    saveSessions(sessions.value);
    currentSessionId.value = session.id;
    return session;
  }

  function switchSession(id) {
    currentSessionId.value = id;
  }

  function deleteSession(id) {
    const idx = sessions.value.findIndex(s => s.id === id);
    if (idx !== -1) {
      sessions.value.splice(idx, 1);
      saveSessions(sessions.value);
    }
    if (currentSessionId.value === id) {
      currentSessionId.value = sessions.value[0]?.id || null;
    }
  }

  function updateSessionTitle(id, title) {
    const session = sessions.value.find(s => s.id === id);
    if (session) {
      session.title = title;
      session.updatedAt = Date.now();
      saveSessions(sessions.value);
    }
  }

  // ===== 消息操作 =====
  function addUserMessage(content) {
    if (!currentSession.value) createSession();

    const msg = {
      id: generateId(),
      role: 'user',
      content,
      timestamp: Date.now(),
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();

    // 自动更新标题
    if (currentSession.value.messages.length === 1) {
      const title = content.slice(0, 30) + (content.length > 30 ? '...' : '');
      updateSessionTitle(currentSession.value.id, title);
    }

    saveSessions(sessions.value);
    return msg;
  }

  function addAIMessage() {
    if (!currentSession.value) return null;

    const msg = {
      id: generateId(),
      role: 'assistant',
      content: '',
      timestamp: Date.now(),
      streaming: true,
    };
    currentSession.value.messages.push(msg);
    currentSession.value.updatedAt = Date.now();
    saveSessions(sessions.value);
    return msg;
  }

  function appendToLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === 'assistant') {
      lastMsg.content += content;
      saveSessions(sessions.value);
    }
  }

  function finishLastAI(content) {
    if (!currentSession.value) return;
    const msgs = currentSession.value.messages;
    const lastMsg = msgs[msgs.length - 1];
    if (lastMsg && lastMsg.role === 'assistant') {
      if (content !== undefined) lastMsg.content = content;
      lastMsg.streaming = false;
      saveSessions(sessions.value);
    }
  }

  function clearCurrentMessages() {
    if (!currentSession.value) return;
    currentSession.value.messages = [];
    saveSessions(sessions.value);
  }

  function ensureSession() {
    if (!currentSessionId.value || !currentSession.value) {
      createSession();
    }
    return currentSession.value;
  }

  // ===== 配置操作 =====
  function setApiBase(base) {
    apiBase.value = base;
    api.setApiBaseUrl(base);
    localStorage.setItem('icooclaw_api_base', base);
  }

  function setWsUrl(url) {
    wsUrl.value = url;
    localStorage.setItem('icooclaw_ws_url', url);
  }

  function setUserId(id) {
    userId.value = id;
    localStorage.setItem('icooclaw_user_id', id);
  }

  // ===== 初始化 =====
  // 确保有一个会话
  if (sessions.value.length === 0) {
    createSession('新对话');
  } else if (!currentSessionId.value) {
    currentSessionId.value = sessions.value[0].id;
  }

  return {
    // 状态
    sessions,
    currentSessionId,
    isLoading,
    apiBase,
    wsUrl,
    userId,
    // 计算属性
    currentSession,
    currentMessages,
    // 会话操作
    createSession,
    switchSession,
    deleteSession,
    updateSessionTitle,
    // 消息操作
    addUserMessage,
    addAIMessage,
    appendToLastAI,
    finishLastAI,
    clearCurrentMessages,
    ensureSession,
    // 配置操作
    setApiBase,
    setWsUrl,
    setUserId,
  };
});
