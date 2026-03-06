// API 服务 - 对接后端 REST API
// 文档: docs/API.md

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

// ===== 通用分页请求 =====

function createPageRequest(page = 1, size = 10) {
  return { page: { page, size } };
}

// ===== Health API =====

export async function checkHealth() {
  return request('/api/v1/health');
}

// ===== Session API =====

export async function getSessionsPage(params = {}) {
  return request('/api/v1/sessions/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || '',
      channel: params.channel || '',
      user_id: params.user_id || '',
    }),
  });
}

// 获取所有会话列表（简化版）
export async function getSessions(params = {}) {
  const response = await getSessionsPage({ page: 1, size: 100, ...params });
  // 后端返回格式: { code, message, data: { page, records: [...] } }
  const data = response.data || response;
  return data?.records || [];
}

export async function getSession(id) {
  return request('/api/v1/sessions/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function createSession(data) {
  return request('/api/v1/sessions/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateSession(data) {
  return request('/api/v1/sessions/save', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteSession(id) {
  return request('/api/v1/sessions/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

// ===== Message API =====

export async function getMessagesPage(params = {}) {
  return request('/api/v1/messages/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      session_id: Number(params.session_id) || 0,
      role: params.role || '',
    }),
  });
}

// 获取会话的所有消息（简化版）
export async function getSessionMessages(sessionId, params = {}) {
  const response = await getMessagesPage({ session_id: sessionId, page: 1, size: 100, ...params });
  // 后端返回格式: { code, message, data: { page, records: [...] } }
  const data = response.data || response;
  return data?.records || [];
}

export async function createMessage(data) {
  return request('/api/v1/messages/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateMessage(data) {
  return request('/api/v1/messages/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteMessage(id) {
  return request('/api/v1/messages/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function getMessage(id) {
  return request('/api/v1/messages/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

// ===== Provider API =====

export async function getProvidersPage(params = {}) {
  return request('/api/v1/providers/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || '',
      enabled: params.enabled,
    }),
  });
}

export async function getProviders() {
  return request('/api/v1/providers/all');
}

export async function getEnabledProviders() {
  return request('/api/v1/providers/enabled');
}

export async function getProvider(id) {
  return request('/api/v1/providers/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function createProvider(data) {
  return request('/api/v1/providers/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateProvider(data) {
  return request('/api/v1/providers/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteProvider(id) {
  return request('/api/v1/providers/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

// 设置 AI Agent 默认模型
export async function setDefaultModel(data) {
  return request('/api/v1/params/default-model/set', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

// 获取 AI Agent 默认模型
export async function getDefaultModel() {
  return request('/api/v1/params/default-model/get');
}

// ===== Skill API =====

export async function getSkillsPage(params = {}) {
  return request('/api/v1/skills/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || '',
      enabled: params.enabled,
      source: params.source || '',
    }),
  });
}

export async function getSkills() {
  return request('/api/v1/skills/all');
}

export async function getEnabledSkills() {
  return request('/api/v1/skills/enabled');
}

export async function getSkill(id) {
  return request('/api/v1/skills/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function getSkillByName(name) {
  return request('/api/v1/skills/get-by-name', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
}

export async function createSkill(data) {
  return request('/api/v1/skills/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateSkill(data) {
  return request('/api/v1/skills/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function upsertSkill(data) {
  return request('/api/v1/skills/upsert', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteSkill(id) {
  return request('/api/v1/skills/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

// ===== MCP API =====

export async function getMCPPage(params = {}) {
  return request('/api/v1/mcp/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || '',
    }),
  });
}

export async function getMCPs() {
  return request('/api/v1/mcp/all');
}

export async function getMCP(id) {
  return request('/api/v1/mcp/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function createMCP(data) {
  return request('/api/v1/mcp/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateMCP(data) {
  return request('/api/v1/mcp/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteMCP(id) {
  return request('/api/v1/mcp/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

// ===== Memory API =====

export async function getMemoriesPage(params = {}) {
  return request('/api/v1/memories/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      type: params.type || '',
      key_word: params.key_word || '',
      user_id: params.user_id || '',
      session_id: params.session_id,
    }),
  });
}

export async function createMemory(data) {
  return request('/api/v1/memories/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateMemory(data) {
  return request('/api/v1/memories/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteMemory(id) {
  return request('/api/v1/memories/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function getMemory(id) {
  return request('/api/v1/memories/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function pinMemory(id) {
  return request('/api/v1/memories/pin', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function unpinMemory(id) {
  return request('/api/v1/memories/unpin', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function softDeleteMemory(id) {
  return request('/api/v1/memories/soft-delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function restoreMemory(id) {
  return request('/api/v1/memories/restore', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function searchMemories(query) {
  return request('/api/v1/memories/search', {
    method: 'POST',
    body: JSON.stringify({ query }),
  });
}

// ===== Task API =====

export async function getTasksPage(params = {}) {
  return request('/api/v1/tasks/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || '',
      enabled: params.enabled,
    }),
  });
}

export async function getTasks() {
  return request('/api/v1/tasks/all');
}

export async function getEnabledTasks() {
  return request('/api/v1/tasks/enabled');
}

export async function getTask(id) {
  return request('/api/v1/tasks/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function createTask(data) {
  return request('/api/v1/tasks/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateTask(data) {
  return request('/api/v1/tasks/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteTask(id) {
  return request('/api/v1/tasks/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function toggleTask(id) {
  return request('/api/v1/tasks/toggle', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

// ===== Channel API =====

export async function getChannelsPage(params = {}) {
  return request('/api/v1/channels/page', {
    method: 'POST',
    body: JSON.stringify({
      ...createPageRequest(params.page, params.size),
      key_word: params.key_word || '',
      enabled: params.enabled,
    }),
  });
}

export async function getChannels() {
  return request('/api/v1/channels/all');
}

export async function getEnabledChannels() {
  return request('/api/v1/channels/enabled');
}

export async function getChannel(id) {
  return request('/api/v1/channels/get', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

export async function createChannel(data) {
  return request('/api/v1/channels/create', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function updateChannel(data) {
  return request('/api/v1/channels/update', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

export async function deleteChannel(id) {
  return request('/api/v1/channels/delete', {
    method: 'POST',
    body: JSON.stringify({ id: Number(id) }),
  });
}

// ===== Config API =====

export async function getConfig() {
  return request('/api/v1/config/');
}

export async function updateConfig(config) {
  return request('/api/v1/config/update', {
    method: 'POST',
    body: JSON.stringify({ config }),
  });
}

export async function overwriteConfig(content) {
  return request('/api/v1/config/overwrite', {
    method: 'POST',
    body: JSON.stringify({ content }),
  });
}

export async function getConfigFile() {
  return request('/api/v1/config/file');
}

export async function getConfigJSON() {
  return request('/api/v1/config/json');
}

// ===== Workspace API =====

export async function getWorkspace() {
  return request('/api/v1/workspace/');
}

export async function setWorkspace(workspace) {
  return request('/api/v1/workspace/set', {
    method: 'POST',
    body: JSON.stringify({ workspace }),
  });
}

// ===== Chat API (WebSocket 流式) =====

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

// ===== API Base URL 管理 =====

export function getApiBaseUrl() {
  return getApiBase();
}

export function setApiBaseUrl(base) {
  setApiBase(base);
}

// 导出默认对象
export default {
  // 通用
  getApiBaseUrl,
  setApiBaseUrl,
  checkHealth,

  // Session
  getSessionsPage,
  getSessions,
  getSession,
  createSession,
  updateSession,
  deleteSession,

  // Message
  getMessagesPage,
  getSessionMessages,
  createMessage,
  updateMessage,
  deleteMessage,
  getMessage,
  
  // Provider
  getProvidersPage,
  getProviders,
  getEnabledProviders,
  getProvider,
  createProvider,
  updateProvider,
  deleteProvider,
  setDefaultModel,
  getDefaultModel,

  // Skill
  getSkillsPage,
  getSkills,
  getEnabledSkills,
  getSkill,
  getSkillByName,
  createSkill,
  updateSkill,
  upsertSkill,
  deleteSkill,
  
  // MCP
  getMCPPage,
  getMCPs,
  getMCP,
  createMCP,
  updateMCP,
  deleteMCP,
  
  // Memory
  getMemoriesPage,
  createMemory,
  updateMemory,
  deleteMemory,
  getMemory,
  pinMemory,
  unpinMemory,
  softDeleteMemory,
  restoreMemory,
  searchMemories,
  
  // Task
  getTasksPage,
  getTasks,
  getEnabledTasks,
  getTask,
  createTask,
  updateTask,
  deleteTask,
  toggleTask,
  
  // Channel
  getChannelsPage,
  getChannels,
  getEnabledChannels,
  getChannel,
  createChannel,
  updateChannel,
  deleteChannel,
  
  // Config
  getConfig,
  updateConfig,
  overwriteConfig,
  getConfigFile,
  getConfigJSON,
  
  // Workspace
  getWorkspace,
  setWorkspace,
  
  // Chat
  sendChatMessage,
  sendChatStream,
};