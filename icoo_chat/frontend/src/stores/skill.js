// Skill Store - 管理技能

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import api from '../services/api';

/**
 * 技能状态管理
 * 提供技能的增删改查、批量操作、导入导出等功能
 */
export const useSkillStore = defineStore('skill', () => {
  // ===== 状态 =====
  const skills = ref([]);
  const loading = ref(false);
  const error = ref(null);
  const tags = ref([]);
  const lastFetchTime = ref(null);

  // ===== 计算属性 =====
  const skillCount = computed(() => skills.value.length);

  const enabledSkills = computed(() =>
    skills.value.filter(s => s.enabled)
  );

  const builtinSkills = computed(() =>
    skills.value.filter(s => s.source === 'builtin')
  );

  const userSkills = computed(() =>
    skills.value.filter(s => s.source === 'workspace' || s.source === 'user')
  );

  const skillsByTag = computed(() => {
    const map = {};
    tags.value.forEach(tag => {
      map[tag] = skills.value.filter(s => s.tags?.includes(tag));
    });
    return map;
  });

  // ===== 基础操作 =====

  /**
   * 获取所有技能
   */
  async function fetchSkills() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getSkills();
      skills.value = response.data || [];
      lastFetchTime.value = new Date();
    } catch (e) {
      error.value = e.message;
      skills.value = [];
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取启用的技能
   */
  async function fetchEnabledSkills() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getEnabledSkills();
      skills.value = response.data || [];
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 分页获取技能
   * @param {Object} params - 查询参数
   */
  async function fetchSkillsPage(params = {}) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getSkillsPage(params);
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 根据ID获取技能
   * @param {string} id - 技能ID
   */
  async function fetchSkillById(id) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getSkillById(id);
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 根据名称获取技能
   * @param {string} name - 技能名称
   */
  async function fetchSkillByName(name) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getSkillByName(name);
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 创建技能
   * @param {Object} skillData - 技能数据
   */
  async function createSkill(skillData) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.createSkill(skillData);
      skills.value.push(response.data);
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 更新技能
   * @param {Object} skillData - 技能数据
   */
  async function updateSkill(skillData) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.updateSkill(skillData);
      const idx = skills.value.findIndex(s => s.id === skillData.id);
      if (idx !== -1) {
        skills.value[idx] = response.data;
      }
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 创建或更新技能
   * @param {Object} skillData - 技能数据
   */
  async function upsertSkill(skillData) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.upsertSkill(skillData);
      await fetchSkills();
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 删除技能
   * @param {string} id - 技能ID
   */
  async function deleteSkill(id) {
    loading.value = true;
    error.value = null;
    try {
      await api.deleteSkill(id);
      skills.value = skills.value.filter(s => s.id !== id);
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 切换技能启用状态
   * @param {string} id - 技能ID
   */
  async function toggleSkill(id) {
    const skill = skills.value.find(s => s.id === id);
    if (skill) {
      return updateSkill({ ...skill, enabled: !skill.enabled });
    }
  }

  /**
   * 根据名称获取技能（本地缓存）
   * @param {string} name - 技能名称
   */
  function getSkillByName(name) {
    return skills.value.find(s => s.name === name);
  }

  /**
   * 根据ID获取技能（本地缓存）
   * @param {string} id - 技能ID
   */
  function getSkillById(id) {
    return skills.value.find(s => s.id === id);
  }

  /**
   * 搜索技能
   * @param {string} keyword - 搜索关键字
   */
  function searchSkills(keyword) {
    if (!keyword) return skills.value;
    const lower = keyword.toLowerCase();
    return skills.value.filter(s =>
      s.name?.toLowerCase().includes(lower) ||
      s.description?.toLowerCase().includes(lower) ||
      s.tags?.some(t => t.toLowerCase().includes(lower))
    );
  }

  // ===== 批量操作 =====

  /**
   * 批量删除技能
   * @param {string[]} ids - 技能ID列表
   */
  async function batchDeleteSkills(ids) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.batchDeleteSkills(ids);
      await fetchSkills();
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 批量更新启用状态
   * @param {string[]} ids - 技能ID列表
   * @param {boolean} enabled - 启用状态
   */
  async function batchUpdateEnabled(ids, enabled) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.batchUpdateSkillsEnabled(ids, enabled);
      await fetchSkills();
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 批量更新始终加载状态
   * @param {string[]} ids - 技能ID列表
   * @param {boolean} alwaysLoad - 始终加载状态
   */
  async function batchUpdateAlwaysLoad(ids, alwaysLoad) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.batchUpdateSkillsAlwaysLoad(ids, alwaysLoad);
      await fetchSkills();
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  // ===== 标签操作 =====

  /**
   * 获取所有标签
   */
  async function fetchTags() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getSkillTags();
      tags.value = response.data || [];
      return tags.value;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 根据标签获取技能
   * @param {string} tag - 标签
   */
  async function fetchSkillsByTag(tag) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getSkillsByTag(tag);
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  // ===== 导入导出 =====

  /**
   * 导出技能
   */
  async function exportSkills() {
    loading.value = true;
    error.value = null;
    try {
      const blob = await api.exportSkills();
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `skills_export_${new Date().toISOString().slice(0, 10)}.json`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 导入技能
   * @param {File} file - 导入文件
   * @param {boolean} overwrite - 是否覆盖
   */
  async function importSkills(file, overwrite = false) {
    loading.value = true;
    error.value = null;
    try {
      const text = await file.text();
      const response = await api.importSkills(text, overwrite);
      await fetchSkills();
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  // ===== 工具函数 =====

  /**
   * 清除错误状态
   */
  function clearError() {
    error.value = null;
  }

  /**
   * 刷新技能列表
   */
  async function refresh() {
    return fetchSkills();
  }

  return {
    // 状态
    skills,
    loading,
    error,
    tags,
    lastFetchTime,
    // 计算属性
    skillCount,
    enabledSkills,
    builtinSkills,
    userSkills,
    skillsByTag,
    // 基础操作
    fetchSkills,
    fetchEnabledSkills,
    fetchSkillsPage,
    fetchSkillById,
    fetchSkillByName,
    createSkill,
    updateSkill,
    upsertSkill,
    deleteSkill,
    toggleSkill,
    getSkillByName,
    getSkillById,
    searchSkills,
    // 批量操作
    batchDeleteSkills,
    batchUpdateEnabled,
    batchUpdateAlwaysLoad,
    // 标签操作
    fetchTags,
    fetchSkillsByTag,
    // 导入导出
    exportSkills,
    importSkills,
    // 工具函数
    clearError,
    refresh,
  };
});
