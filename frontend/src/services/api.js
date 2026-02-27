// API 服务 - 对接后端 REST API

const API_BASE_KEY = 'icooclaw_api_base';

function getApiBase() {
  const stored = localStorage.getItem(API_BASE_KEY);
  return stored || 'http://localhost:8080';
}

function setApiBase(base) {
  localStorage.setItem(API_BASE_KEY, base);
}

function getHeaders() {
  return {
    'Content-Type': 'application/json',
  };
}

async function request(endpoint, options = {}) {
  const base = getApiBase();
  const url = `${base}${endpoint}`;

  const config = {
    ...options,
    headers: {
      ...getHeaders(),
      ...options.headers,
    },
  };

  try {
    const response = await fetch(url, config);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    return await response.json();
  } catch (error) {
    console.error(`API request failed: ${endpoint}`, error);
    throw error;
  }
}

// ===== API 接口 =====

// 健康检查
export async function checkHealth() {
  return request('/api/v1/health');
}

// 获取 Provider 列表
export async function getProviders() {
  return request('/api/v1/providers');
}

// 获取会话列表
export async function getSessions() {
  return request('/api/v1/sessions');
}

// 获取单个会话
export async function getSession(id) {
  return request(`/api/v1/sessions/${id}`);
}

// 获取会话消息
export async function getSessionMessages(id) {
  return request(`/api/v1/sessions/${id}/messages`);
}

// 创建会话
export async function createSession(data) {
  return request('/api/v1/sessions', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// 更新会话
export async function updateSession(id, data) {
  return request(`/api/v1/sessions/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

// 删除会话
export async function deleteSession(id) {
  return request(`/api/v1/sessions/${id}`, {
    method: 'DELETE',
  });
}

// 发送聊天消息 (非流式)
export async function sendChatMessage(content, chatId, userId) {
  return request('/api/v1/chat', {
    method: 'POST',
    body: JSON.stringify({
      content,
      chat_id: chatId,
      user_id: userId,
    }),
  });
}

// 发送流式聊天消息 (使用 SSE)
export async function sendChatStream(content, chatId, userId) {
  const base = getApiBase();
  const url = `${base}/api/v1/chat/stream`;

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      content,
      chat_id: chatId,
      user_id: userId,
    }),
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  if (!response.body) {
    throw new Error('No response body');
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  async function* streamReader() {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // 处理 SSE 格式: data: content\n\n
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6).trim();
          if (data === '[DONE]') {
            return;
          }
          yield data;
        }
      }
    }
  }

  return streamReader();
}

// ===== Skill API =====

// 获取所有技能
export async function getSkills() {
  return request('/api/v1/skills', { method: 'GET' });
}

// 获取单个技能
export async function getSkill(id) {
  return request(`/api/v1/skills/${id}`, { method: 'GET' });
}

// 创建技能
export async function createSkill(data) {
  return request('/api/v1/skills', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// 更新技能
export async function updateSkill(id, data) {
  return request(`/api/v1/skills/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

// 删除技能
export async function deleteSkill(id) {
  return request(`/api/v1/skills/${id}`, {
    method: 'DELETE',
  });
}

// 获取/设置 API 基础地址
export function getApiBaseUrl() {
  return getApiBase();
}

export function setApiBaseUrl(base) {
  setApiBase(base);
}

// 导出默认配置
export default {
  getApiBase,
  setApiBase,
  checkHealth,
  getProviders,
  getSessions,
  getSession,
  getSessionMessages,
  createSession,
  updateSession,
  deleteSession,
  sendChatMessage,
  sendChatStream,
  getApiBaseUrl,
  setApiBaseUrl,
  getSkills,
  getSkill,
  createSkill,
  updateSkill,
  deleteSkill,
};
