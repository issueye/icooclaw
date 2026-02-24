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
    skills.value.filter(s => s.source === 'user')
  );

  // ===== 操作 =====
  async function fetchSkills() {
    loading.value = true;
    error.value = null;
    try {
      const data = await api.getSkills();
      skills.value = data.skills || [];
    } catch (e) {
      error.value = e.message;
      skills.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function createSkill(skillData) {
    loading.value = true;
    error.value = null;
    try {
      const newSkill = await api.createSkill(skillData);
      skills.value.push(newSkill);
      return newSkill;
    } catch (e) {
      error.value = e.message;
      throw e;
    } finally {
      loading.value = false;
    }
  }

  async function updateSkill(id, skillData) {
    loading.value = true;
    error.value = null;
    try {
      const updated = await api.updateSkill(id, skillData);
      const idx = skills.value.findIndex(s => s.id === id);
      if (idx !== -1) {
        skills.value[idx] = updated;
      }
      return updated;
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
      return updateSkill(id, { ...skill, enabled: !skill.enabled });
    }
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
    createSkill,
    updateSkill,
    deleteSkill,
    toggleSkill,
  };
});
