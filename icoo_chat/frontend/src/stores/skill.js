// Skill Store - 管理技能

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import api from '../services/api';

export const useSkillStore = defineStore('skill', () => {
  // ===== 状态 =====
  const skills = ref([]);
  const loading = ref(false);
  const error = ref(null);

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

  // ===== 操作 =====
  async function fetchSkills() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getSkills();
      // API 返回格式: { code, message, data: [...] }
      skills.value = response.data || [];
    } catch (e) {
      error.value = e.message;
      skills.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function fetchEnabledSkills() {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.getEnabledSkills();
      skills.value = response.data || [];
    } finally {
      loading.value = false;
    }
  }

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

  async function upsertSkill(skillData) {
    loading.value = true;
    error.value = null;
    try {
      const response = await api.upsertSkill(skillData);
      // 刷新列表
      await fetchSkills();
      return response.data;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

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

  async function toggleSkill(id) {
    const skill = skills.value.find(s => s.id === id);
    if (skill) {
      return updateSkill({ ...skill, enabled: !skill.enabled });
    }
  }

  function getSkillByName(name) {
    return skills.value.find(s => s.name === name);
  }

  return {
    // 状态
    skills,
    loading,
    error,
    // 计算属性
    skillCount,
    enabledSkills,
    builtinSkills,
    userSkills,
    // 操作
    fetchSkills,
    fetchEnabledSkills,
    fetchSkillsPage,
    createSkill,
    updateSkill,
    upsertSkill,
    deleteSkill,
    toggleSkill,
    getSkillByName,
  };
});