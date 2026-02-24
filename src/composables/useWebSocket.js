import { ref, onUnmounted } from "vue";

// WebSocket 连接状态常量
export const WS_STATUS = {
  CONNECTING: "connecting",
  CONNECTED: "connected",
  DISCONNECTED: "disconnected",
  ERROR: "error",
};

export function useWebSocket() {
  const ws = ref(null);
  const status = ref(WS_STATUS.DISCONNECTED);
  const lastError = ref(null);

  let reconnectTimer = null;
  let reconnectAttempts = 0;
  const MAX_RECONNECT_ATTEMPTS = 10;
  const messageHandlers = [];
  let currentUrl = "";

  function connect(url) {
    if (!url) return;
    currentUrl = url;

    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.close();
    }

    status.value = WS_STATUS.CONNECTING;
    lastError.value = null;

    try {
      ws.value = new WebSocket(url);

      ws.value.onopen = () => {
        status.value = WS_STATUS.CONNECTED;
        reconnectAttempts = 0;
        lastError.value = null;
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
          console.error("解析 WebSocket 消息失败:", e, event.data);
        }
      };

      ws.value.onerror = (event) => {
        status.value = WS_STATUS.ERROR;
        lastError.value = "连接错误";
      };

      ws.value.onclose = (event) => {
        status.value = WS_STATUS.DISCONNECTED;
        scheduleReconnect();
      };
    } catch (e) {
      status.value = WS_STATUS.ERROR;
      lastError.value = e.message;
      scheduleReconnect();
    }
  }

  function scheduleReconnect() {
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) return;
    clearTimeout(reconnectTimer);
    const delay = Math.min(1000 * Math.pow(1.5, reconnectAttempts), 30000);
    reconnectAttempts++;
    reconnectTimer = setTimeout(() => {
      if (currentUrl) connect(currentUrl);
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
    clearTimeout(reconnectTimer);
    reconnectAttempts = MAX_RECONNECT_ATTEMPTS; // 防止重连
    if (ws.value) {
      ws.value.close();
      ws.value = null;
    }
    status.value = WS_STATUS.DISCONNECTED;
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
    connect,
    disconnect,
    send,
    onMessage,
  };
}
