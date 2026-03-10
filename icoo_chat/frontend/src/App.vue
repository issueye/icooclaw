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
    <main class="main-content">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { RouterView } from "vue-router";
import { useThemeStore } from "./stores/theme";

const themeStore = useThemeStore();
themeStore.initTheme();

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

.main-content {
  flex: 1;
  overflow: hidden;
  width: 100%;
  height: calc(100% - var(--header-height));
}
</style>
