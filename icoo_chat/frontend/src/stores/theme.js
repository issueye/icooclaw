import { defineStore } from "pinia";
import { ref, watch } from "vue";

export const useThemeStore = defineStore("theme", () => {
  const theme = ref(localStorage.getItem("theme") || "dark");

  const setTheme = (newTheme) => {
    theme.value = newTheme;
    localStorage.setItem("theme", newTheme);
    applyTheme(newTheme);
  };

  const toggleTheme = () => {
    const newTheme = theme.value === "dark" ? "light" : "dark";
    setTheme(newTheme);
  };

  const applyTheme = (themeName) => {
    document.documentElement.setAttribute("data-theme", themeName);
  };

  const initTheme = () => {
    applyTheme(theme.value);
  };

  return {
    theme,
    setTheme,
    toggleTheme,
    initTheme,
  };
});
