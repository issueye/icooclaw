// Provider Store - 管理 LLM Provider 信息

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import api from '../services/api';

export const useProviderStore = defineStore('provider', () => {
  // ===== 状态 =====
  const providers = ref([]);
  const currentProvider = ref(null);
  const loading = ref(false);
  const error = ref(null);

  // ===== 计算属性 =====
  const providerCount = computed(() => providers.value.length);

  const providerNames = computed(() =>
    providers.value.map(p => p.name)
  );

  const enabledProviders = computed(() =>
    providers.value.filter(p => p.enabled)
  );

  // ===== 操作 =====
  async function fetchProviders() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getProviders();
      // API 返回格式: { code, message, data: [...] }
      providers.value = response.data || [];
    } catch (e) {
      error.value = e.message;
      providers.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function fetchEnabledProviders() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getEnabledProviders();
      providers.value = response.data || [];
    } catch (e) {
      error.value = e.message;
      providers.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function fetchProvidersPage(params = {}) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getProvidersPage(params);
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function createProvider(providerData) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.createProvider(providerData);
      providers.value.push(response.data);
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function updateProvider(providerData) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.updateProvider(providerData);
      const idx = providers.value.findIndex(p => p.id === providerData.id);
      if (idx !== -1) {
        providers.value[idx] = response.data;
      }
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function deleteProvider(id) {
    loading.value = true;
    error.value = null;
    try {
      await api.deleteProvider(id);
      providers.value = providers.value.filter(p => p.id !== id);
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  function setCurrentProvider(name) {
    currentProvider.value = name;
  }

  function getProviderByName(name) {
    return providers.value.find(p => p.name === name);
  }

  return {
    // 状态
    providers,
    currentProvider,
    loading,
    error,
    // 计算属性
    providerCount,
    providerNames,
    enabledProviders,
    // 操作
    fetchProviders,
    fetchEnabledProviders,
    fetchProvidersPage,
    createProvider,
    updateProvider,
    deleteProvider,
    setCurrentProvider,
    getProviderByName,
  };
});