<template>
  <div class="app-container">
    <header class="custom-header">
      <div class="header-drag-region">
        <span class="app-title">icoo_chat</span>
      </div>
      <div class="window-controls">
        <button class="control-btn minimize-btn" @click="handleMinimize">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <rect x="1" y="5.5" width="10" height="1" fill="currentColor"/>
          </svg>
        </button>
        <button class="control-btn close-btn" @click="handleClose">
          <svg width="12" height="12" viewBox="0 0 12 12">
            <path d="M1 1L11 11M11 1L1 11" stroke="currentColor" stroke-width="1.5" fill="none"/>
          </svg>
        </button>
      </div>
    </header>
    <div class="app-body">
      <aside class="sidebar">
        <nav class="sidebar-nav">
          <router-link 
            v-for="item in menuItems" 
            :key="item.path"
            :to="item.path" 
            class="nav-item"
            :class="{ active: isActive(item.path) }"
          >
            <component :is="item.icon" :size="20" />
            <span class="nav-label">{{ item.label }}</span>
          </router-link>
        </nav>
      </aside>
      <main class="main-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup>
import { RouterView, useRoute } from "vue-router";
import { useThemeStore } from "./stores/theme";
import { MessageSquare, Clock, Settings } from "lucide-vue-next";

const themeStore = useThemeStore();
themeStore.initTheme();

const menuItems = [
  { path: '/', label: '聊天', icon: MessageSquare },
  { path: '/tasks', label: '定时任务', icon: Clock },
  { path: '/settings', label: '设置', icon: Settings },
];

const route = useRoute();

function isActive(path) {
  if (path === '/') {
    return route.path === '/';
  }
  return route.path.startsWith(path);
}

function handleMinimize() {
  window.go.services.App.MinimizeWindow();
}

function handleClose() {
  window.go.services.App.CloseWindow();
}
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  overflow: hidden;
  background: var(--color-bg-primary);
}

.custom-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--header-height);
  background: #1e293b;
  color: #fff;
  user-select: none;
}

.header-drag-region {
  flex: 1;
  display: flex;
  align-items: center;
  padding-left: 16px;
  height: 100%;
  --wails-draggable: drag;
}

.app-title {
  font-size: 14px;
  font-weight: 500;
}

.window-controls {
  display: flex;
  -webkit-app-region: no-drag;
}

.control-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: var(--header-height);
  border: none;
  background: transparent;
  color: #fff;
  cursor: pointer;
  transition: background-color 0.15s;
}

.control-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

.close-btn:hover {
  background: #e81123;
}

.app-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 60px;
  background: #1e293b;
  border-right: 1px solid #334155;
  display: flex;
  flex-direction: column;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  padding: 8px 0;
}

.nav-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 10px 4px;
  color: #94a3b8;
  text-decoration: none;
  transition: all 0.15s;
  gap: 4px;
}

.nav-item:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.1);
}

.nav-item.active {
  color: #7c3aed;
  background: rgba(124, 58, 237, 0.1);
  border-left: 2px solid #7c3aed;
}

.nav-label {
  font-size: 10px;
  font-weight: 500;
}

.main-content {
  flex: 1;
  overflow: hidden;
}
</style>
