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

  // ===== 操作 =====
  async function fetchProviders() {
    loading.value = true;
    error.value = null;
    try {
      const data = await api.getProviders();
      providers.value = data.providers || [];
    } catch (e) {
      error.value = e.message;
      providers.value = [];
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
    // 操作
    fetchProviders,
    setCurrentProvider,
    getProviderByName,
  };
});
