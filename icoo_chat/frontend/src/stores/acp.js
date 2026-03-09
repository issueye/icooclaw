// ACP 状态管理
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import * as acp from '@/services/acp';

export const useACPStore = defineStore('acp', () => {
  // 状态
  const connected = ref(false);
  const connecting = ref(false);
  const config = ref(acp.getACPConfig());
  const agents = ref([]);
  const currentAgent = ref(null);
  const currentSession = ref(null);
  const messages = ref([]);
  const loading = ref(false);

  // 计算属性
  const isConnected = computed(() => connected.value);
  const hasAgent = computed(() => currentAgent.value !== null);
  const hasSession = computed(() => currentSession.value !== null);

  // 初始化
  async function init() {
    const status = await acp.getACPStatus();
    connected.value = status.connected || false;
    if (status.config) {
      config.value = { ...config.value, ...status.config };
    }
    await refreshAgents();
  }

  // 连接
  async function connect() {
    if (connecting.value || connected.value) return;

    connecting.value = true;
    try {
      await acp.initACP(config.value.endpoint, config.value.apiKey, config.value.aid);
      const result = await acp.connectACP();
      if (result === 'OK') {
        connected.value = true;
        await refreshAgents();
      }
    } catch (error) {
      console.error('Failed to connect:', error);
      throw error;
    } finally {
      connecting.value = false;
    }
  }

  // 断开
  async function disconnect() {
    try {
      await acp.disconnectACP();
    } catch (error) {
      console.error('Failed to disconnect:', error);
    } finally {
      connected.value = false;
      agents.value = [];
      currentAgent.value = null;
      currentSession.value = null;
      messages.value = [];
    }
  }

  // 更新配置
  function updateConfig(newConfig) {
    config.value = { ...config.value, ...newConfig };
    acp.saveACPConfig(config.value);
  }

  // 刷新 Agent 列表
  async function refreshAgents() {
    try {
      const list = await acp.listConnectedAgents();
      agents.value = list || [];
    } catch (error) {
      console.error('Failed to refresh agents:', error);
    }
  }

  // 连接 Agent
  async function connectAgent(aid) {
    try {
      const agent = await acp.connectAgent(aid);
      if (agent) {
        currentAgent.value = agent;
        await refreshAgents();
      }
      return agent;
    } catch (error) {
      console.error('Failed to connect agent:', error);
      throw error;
    }
  }

  // 断开 Agent
  async function disconnectAgent(aid) {
    try {
      await acp.disconnectAgent(aid);
      if (currentAgent.value?.aid === aid) {
        currentAgent.value = null;
        currentSession.value = null;
        messages.value = [];
      }
      await refreshAgents();
    } catch (error) {
      console.error('Failed to disconnect agent:', error);
      throw error;
    }
  }

  // 创建会话
  async function createSession() {
    if (!currentAgent.value) {
      throw new Error('No agent selected');
    }
    try {
      const sessionId = await acp.createACPSession(currentAgent.value.aid);
      currentSession.value = sessionId;
      messages.value = [];
      return sessionId;
    } catch (error) {
      console.error('Failed to create session:', error);
      throw error;
    }
  }

  // 发送消息
  async function sendMessage(content) {
    if (!currentSession.value) {
      await createSession();
    }

    // 添加用户消息
    messages.value.push({
      id: Date.now(),
      role: 'user',
      content,
      timestamp: new Date().toISOString(),
    });

    loading.value = true;

    try {
      await acp.sendACPMessage(currentSession.value, content);

      // 添加 AI 消息占位
      const aiMsg = {
        id: Date.now() + 1,
        role: 'assistant',
        content: '',
        thinking: '',
        toolCalls: [],
        timestamp: new Date().toISOString(),
      };
      messages.value.push(aiMsg);

      return aiMsg;
    } catch (error) {
      console.error('Failed to send message:', error);
      throw error;
    }
  }

  // 关闭会话
  async function closeSession() {
    if (currentSession.value) {
      try {
        await acp.closeACPSession(currentSession.value);
      } catch (error) {
        console.error('Failed to close session:', error);
      }
      currentSession.value = null;
      messages.value = [];
    }
  }

  // 处理接收到的消息
  function handleMessage(msg) {
    const lastMsg = messages.value[messages.value.length - 1];
    if (!lastMsg || lastMsg.role !== 'assistant') return;

    switch (msg.type) {
      case 'chunk':
      case 'content':
        lastMsg.content += msg.data?.content || '';
        break;
      case 'thinking':
        lastMsg.thinking = msg.data?.content || '';
        break;
      case 'tool_call':
        lastMsg.toolCalls = lastMsg.toolCalls || [];
        lastMsg.toolCalls.push({
          id: msg.data?.tool_call_id,
          name: msg.data?.tool_name,
          arguments: msg.data?.arguments,
          status: 'pending',
        });
        break;
      case 'tool_result':
        const toolCall = lastMsg.toolCalls?.find(t => t.id === msg.data?.tool_call_id);
        if (toolCall) {
          toolCall.status = 'completed';
          toolCall.result = msg.data?.result;
        }
        break;
      case 'end':
        loading.value = false;
        break;
      case 'error':
        lastMsg.content += '\n\n[错误] ' + (msg.error?.message || '未知错误');
        loading.value = false;
        break;
    }
  }

  return {
    // 状态
    connected,
    connecting,
    config,
    agents,
    currentAgent,
    currentSession,
    messages,
    loading,

    // 计算属性
    isConnected,
    hasAgent,
    hasSession,

    // 方法
    init,
    connect,
    disconnect,
    updateConfig,
    refreshAgents,
    connectAgent,
    disconnectAgent,
    createSession,
    sendMessage,
    closeSession,
    handleMessage,
  };
});