import { ref, onUnmounted } from "vue";

// WebSocket 连接状态常量
export const WS_STATUS = {
  CONNECTING: "connecting",
  CONNECTED: "connected",
  DISCONNECTED: "disconnected",
  RECONNECTING: "reconnecting",
  ERROR: "error",
};

export function useWebSocket() {
  const ws = ref(null);
  const status = ref(WS_STATUS.DISCONNECTED);
  const lastError = ref(null);
  const reconnectInfo = ref({ attempts: 0, nextRetryIn: 0 });

  let reconnectTimer = null;
  let reconnectAttempts = 0;
  const MAX_RECONNECT_ATTEMPTS = 10;
  const INITIAL_RECONNECT_DELAY = 1000;
  const MAX_RECONNECT_DELAY = 30000;
  const RECONNECT_BACKOFF = 1.5;
  const messageHandlers = [];
  let currentUrl = "";
  let isIntentionalClose = false;

  function connect(url) {
    if (!url) return;
    currentUrl = url;
    isIntentionalClose = false;

    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.close();
    }

    status.value = reconnectAttempts > 0 ? WS_STATUS.RECONNECTING : WS_STATUS.CONNECTING;
    lastError.value = null;

    try {
      ws.value = new WebSocket(url);

      ws.value.onopen = () => {
        status.value = WS_STATUS.CONNECTED;
        reconnectAttempts = 0;
        reconnectInfo.value = { attempts: 0, nextRetryIn: 0 };
        lastError.value = null;
        console.log("[WebSocket] 连接成功");
      };

      ws.value.onmessage = (event) => {
        try {
          // 支持多行 JSON（后端可能一次性拼接多条消息）
          const lines = event.data.split("\n").filter((l) => l.trim());
          for (const line of lines) {
            const msg = JSON.parse(line);
            messageHandlers.forEach((handler) => handler(msg));
          }
        } catch (e) {
          console.error("[WebSocket] 解析消息失败:", e, event.data);
        }
      };

      ws.value.onerror = (event) => {
        status.value = WS_STATUS.ERROR;
        lastError.value = "连接错误";
        console.error("[WebSocket] 连接错误");
      };

      ws.value.onclose = (event) => {
        status.value = WS_STATUS.DISCONNECTED;
        
        // 如果是故意关闭，不重连
        if (isIntentionalClose) {
          console.log("[WebSocket] 主动关闭连接");
          return;
        }

        // 如果是正常关闭码(1000)，不重连
        if (event.code === 1000) {
          console.log("[WebSocket] 正常关闭");
          return;
        }

        console.warn(`[WebSocket] 连接关闭，code: ${event.code}, reason: ${event.reason || 'unknown'}`);
        scheduleReconnect();
      };
    } catch (e) {
      status.value = WS_STATUS.ERROR;
      lastError.value = e.message;
      console.error("[WebSocket] 创建连接失败:", e.message);
      scheduleReconnect();
    }
  }

  function scheduleReconnect() {
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      status.value = WS_STATUS.ERROR;
      lastError.value = `重连失败，已达到最大重试次数 (${MAX_RECONNECT_ATTEMPTS})`;
      console.error("[WebSocket] 达到最大重试次数，停止重连");
      return;
    }

    clearTimeout(reconnectTimer);
    
    // 指数退避算法
    const delay = Math.min(
      INITIAL_RECONNECT_DELAY * Math.pow(RECONNECT_BACKOFF, reconnectAttempts),
      MAX_RECONNECT_DELAY
    );
    
    reconnectAttempts++;
    reconnectInfo.value = { 
      attempts: reconnectAttempts, 
      nextRetryIn: Math.ceil(delay / 1000)
    };
    
    status.value = WS_STATUS.RECONNECTING;
    
    console.log(`[WebSocket] 准备重连，尝试 ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS}，${Math.ceil(delay/1000)}秒后重试...`);
    
    reconnectTimer = setTimeout(() => {
      if (currentUrl && !isIntentionalClose) connect(currentUrl);
    }, delay);
  }

  function send(data) {
    if (!ws.value || ws.value.readyState !== WebSocket.OPEN) {
      console.warn("WebSocket 未连接，无法发送消息");
      return false;
    }
    ws.value.send(typeof data === "string" ? data : JSON.stringify(data));
    return true;
  }

  function disconnect() {
    isIntentionalClose = true;
    clearTimeout(reconnectTimer);
    reconnectAttempts = MAX_RECONNECT_ATTEMPTS; // 防止重连
    if (ws.value) {
      ws.value.close(1000, "客户端主动关闭");
      ws.value = null;
    }
    status.value = WS_STATUS.DISCONNECTED;
    console.log("[WebSocket] 主动断开连接");
  }

  // 手动重连
  function manualReconnect() {
    reconnectAttempts = 0;
    reconnectInfo.value = { attempts: 0, nextRetryIn: 0 };
    if (currentUrl) {
      connect(currentUrl);
    }
  }

  function onMessage(handler) {
    messageHandlers.push(handler);
    return () => {
      const idx = messageHandlers.indexOf(handler);
      if (idx !== -1) messageHandlers.splice(idx, 1);
    };
  }

  onUnmounted(() => {
    disconnect();
  });

  return {
    status,
    lastError,
    reconnectInfo,
    connect,
    disconnect,
    manualReconnect,
    send,
    onMessage,
  };
}
