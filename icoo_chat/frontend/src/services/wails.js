// Wails 运行时服务
// 用于与 Go 后端通信

import { ref } from 'vue';

// 检测是否在 Wails 环境中运行
export function isWailsEnv() {
  return typeof window !== 'undefined' && window.go !== undefined;
}

// Wails 服务
export const wailsService = {
  // 应用服务引用
  app: null,

  // 初始化
  init() {
    if (isWailsEnv()) {
      // 动态导入 Wails 运行时
      this.app = window.go.services.App;
    }
  },

  // 获取配置
  async getConfig() {
    if (!this.app) return null;
    return await this.app.GetConfig();
  },

  // 设置配置
  async setConfig(config) {
    if (!this.app) return;
    await this.app.SetConfig(config);
  },

  // 发送消息
  async sendMessage(sessionId, content) {
    if (!this.app) return [];
    return await this.app.SendMessage(sessionId, content);
  },

  // 创建会话
  async createSession() {
    if (!this.app) return null;
    return await this.app.CreateSession();
  },

  // 获取 Agent 状态
  async getAgentStatus() {
    if (!this.app) return { connected: false };
    return await this.app.GetAgentStatus();
  },

  // 列出模型
  async listModels() {
    if (!this.app) return [];
    return await this.app.ListModels();
  },

  // 执行工具
  async executeTool(toolName, args) {
    if (!this.app) return null;
    return await this.app.ExecuteTool(toolName, args);
  },
};

// 事件系统
class EventEmitter {
  constructor() {
    this.listeners = new Map();
  }

  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, []);
    }
    this.listeners.get(event).push(callback);
    return () => this.off(event, callback);
  }

  off(event, callback) {
    if (!this.listeners.has(event)) return;
    const callbacks = this.listeners.get(event);
    const index = callbacks.indexOf(callback);
    if (index > -1) {
      callbacks.splice(index, 1);
    }
  }

  emit(event, data) {
    if (!this.listeners.has(event)) return;
    this.listeners.get(event).forEach(callback => callback(data));
  }
}

export const eventEmitter = new EventEmitter();

// Wails 事件名称
export const WailsEvents = {
  MESSAGE_STREAM: 'message:stream',
  MESSAGE_COMPLETE: 'message:complete',
  MESSAGE_ERROR: 'message:error',
  AGENT_STATUS: 'agent:status',
};