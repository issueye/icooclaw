// Wails 聊天 composable
// 处理 Wails 模式下的聊天逻辑

import { ref, onMounted, onUnmounted } from 'vue';
import { isWailsEnv, wailsService, eventEmitter, WailsEvents } from '../services/wails';

export function useWailsChat() {
  const isWails = ref(false);
  const isConnected = ref(false);
  const isStreaming = ref(false);
  const currentContent = ref('');
  const currentThinking = ref('');
  const error = ref(null);

  // 初始化
  onMounted(async () => {
    isWails.value = isWailsEnv();
    if (isWails.value) {
      wailsService.init();
      await checkConnection();
    }
  });

  // 检查连接状态
  async function checkConnection() {
    if (!isWails.value) return;
    try {
      const status = await wailsService.getAgentStatus();
      isConnected.value = status.connected;
      if (!status.connected && status.error) {
        error.value = status.error;
      }
    } catch (e) {
      isConnected.value = false;
      error.value = e.message;
    }
  }

  // 发送消息
  async function sendMessage(sessionId, content, callbacks = {}) {
    if (!isWails.value) {
      console.warn('[WailsChat] Not in Wails environment');
      return;
    }

    isStreaming.value = true;
    currentContent.value = '';
    currentThinking.value = '';
    error.value = null;

    try {
      const events = await wailsService.sendMessage(sessionId, content);

      for (const event of events) {
        switch (event.type) {
          case 'delta':
            if (event.content) {
              currentContent.value += event.content;
              callbacks.onContent?.(event.content);
            }
            if (event.data?.thinking) {
              currentThinking.value = event.data.thinking;
              callbacks.onThinking?.(event.data.thinking);
            }
            break;

          case 'tool_call':
            callbacks.onToolCall?.(event.data);
            break;

          case 'finish':
            callbacks.onFinish?.(event.data);
            break;

          case 'done':
            callbacks.onComplete?.(currentContent.value);
            break;

          case 'error':
            error.value = event.error;
            callbacks.onError?.(event.error);
            break;
        }
      }
    } catch (e) {
      error.value = e.message;
      callbacks.onError?.(e.message);
    } finally {
      isStreaming.value = false;
    }

    return {
      content: currentContent.value,
      thinking: currentThinking.value,
      error: error.value,
    };
  }

  // 取消流式响应
  function cancelStream() {
    isStreaming.value = false;
  }

  return {
    isWails,
    isConnected,
    isStreaming,
    currentContent,
    currentThinking,
    error,
    checkConnection,
    sendMessage,
    cancelStream,
  };
}