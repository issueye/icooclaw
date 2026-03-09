// ACP 协议客户端服务
// 对接后端 ACP 接口

const ACP_STORAGE_KEY = 'icooclaw_acp_config';

// 默认配置
const defaultConfig = {
  endpoint: 'wss://ap.agentunion.cn',
  apiKey: '',
  aid: '',
};

// 获取保存的配置
export function getACPConfig() {
  const stored = localStorage.getItem(ACP_STORAGE_KEY);
  if (stored) {
    try {
      return { ...defaultConfig, ...JSON.parse(stored) };
    } catch (e) {
      console.error('Failed to parse ACP config:', e);
    }
  }
  return defaultConfig;
}

// 保存配置
export function saveACPConfig(config) {
  localStorage.setItem(ACP_STORAGE_KEY, JSON.stringify(config));
}

// 获取 Wails App 实例
function getApp() {
  if (typeof window !== 'undefined' && window.go?.services?.App) {
    return window.go.services.App;
  }
  return null;
}

// 初始化 ACP 客户端
export async function initACP(endpoint, apiKey, aid) {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  return await app.InitACP(endpoint, apiKey, aid);
}

// 连接到 AP
export async function connectACP() {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  return await app.ConnectACP();
}

// 断开连接
export async function disconnectACP() {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  return await app.DisconnectACP();
}

// 获取连接状态
export async function getACPStatus() {
  const app = getApp();
  if (!app) {
    return { connected: false, config: null };
  }
  return await app.GetACPStatus();
}

// 连接 Agent
export async function connectAgent(aid) {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  const result = await app.ConnectAgent(aid);
  if (result && result.startsWith('{')) {
    return JSON.parse(result);
  }
  return null;
}

// 断开 Agent
export async function disconnectAgent(aid) {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  return await app.DisconnectAgent(aid);
}

// 获取 Agent 信息
export async function getAgentInfo(aid) {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  const result = await app.GetAgentInfo(aid);
  if (result && result.startsWith('{')) {
    return JSON.parse(result);
  }
  return null;
}

// 列出已连接 Agent
export async function listConnectedAgents() {
  const app = getApp();
  if (!app) {
    return [];
  }
  const result = await app.ListConnectedAgents();
  if (result && result.startsWith('[')) {
    return JSON.parse(result);
  }
  return [];
}

// 创建会话
export async function createACPSession(agentAID) {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  return await app.CreateACPSession(agentAID);
}

// 发送消息
export async function sendACPMessage(sessionID, content) {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  return await app.SendACPMessage(sessionID, content);
}

// 关闭会话
export async function closeACPSession(sessionID) {
  const app = getApp();
  if (!app) {
    throw new Error('Wails 环境未就绪');
  }
  return await app.CloseACPSession(sessionID);
}

// 导出默认对象
export default {
  getACPConfig,
  saveACPConfig,
  initACP,
  connectACP,
  disconnectACP,
  getACPStatus,
  connectAgent,
  disconnectAgent,
  getAgentInfo,
  listConnectedAgents,
  createACPSession,
  sendACPMessage,
  closeACPSession,
};