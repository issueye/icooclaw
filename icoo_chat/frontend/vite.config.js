import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import path from "path";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      // API 代理 - 将 /api 请求转发到后端
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      // WebSocket 代理
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
      },
    }
  }
})
