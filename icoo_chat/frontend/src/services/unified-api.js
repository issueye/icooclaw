// 统一 API 服务
// 根据运行环境自动选择 Wails 或 HTTP API

import { isWailsEnv, wailsService } from './wails';
import * as httpApi from './api';

// 统一 API 接口
export const api = {
  // 环境检测
  isWails: isWailsEnv(),

  // 初始化
  init() {
    if (this.isWails) {
      wailsService.init();
      console.log('[API] 使用 Wails 运行时');
    } else {
      console.log('[API] 使用 HTTP API');
    }
  },

  // ===== 配置 =====
  async getConfig() {
    if (this.isWails) {
      return wailsService.getConfig();
    }
    return httpApi.getConfig();
  },

  async setConfig(config) {
    if (this.isWails) {
      return wailsService.setConfig(config);
    }
    return httpApi.updateConfig(config);
  },

  // ===== 会话 =====
  async getSessions(params) {
    if (this.isWails) {
      // Wails 模式下使用本地存储
      const sessions = JSON.parse(localStorage.getItem('wails_sessions') || '[]');
      return sessions;
    }
    return httpApi.getSessions(params);
  },

  async createSession(title = '新对话') {
    if (this.isWails) {
      const session = await wailsService.createSession();
      if (session) {
        session.title = title;
        const sessions = JSON.parse(localStorage.getItem('wails_sessions') || '[]');
        sessions.unshift(session);
        localStorage.setItem('wails_sessions', JSON.stringify(sessions));
      }
      return session;
    }
    return httpApi.createSession({ title });
  },

  async deleteSession(id) {
    if (this.isWails) {
      const sessions = JSON.parse(localStorage.getItem('wails_sessions') || '[]');
      const index = sessions.findIndex(s => s.id === id);
      if (index > -1) {
        sessions.splice(index, 1);
        localStorage.setItem('wails_sessions', JSON.stringify(sessions));
      }
      return;
    }
    return httpApi.deleteSession(id);
  },

  // ===== 消息 =====
  async sendMessage(sessionId, content) {
    if (this.isWails) {
      return wailsService.sendMessage(sessionId, content);
    }
    // HTTP 模式返回流式读取器
    return httpApi.sendChatStream(content, sessionId);
  },

  // ===== Agent =====
  async getAgentStatus() {
    if (this.isWails) {
      return wailsService.getAgentStatus();
    }
    return httpApi.checkHealth();
  },

  async listModels() {
    if (this.isWails) {
      return wailsService.listModels();
    }
    return httpApi.getEnabledProviders();
  },

  // ===== 工具 =====
  async executeTool(toolName, args) {
    if (this.isWails) {
      return wailsService.executeTool(toolName, args);
    }
    throw new Error('Tool execution not supported in HTTP mode');
  },

  // ===== 兼容原有 API =====
  ...httpApi,
};

export default api;