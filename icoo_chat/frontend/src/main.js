import { createApp } from "vue";
import { createPinia } from "pinia";
import router from "./router";
import "./style.css";
import App from "@/App.vue";
import { api } from "@/services/unified-api";

// 初始化统一 API
api.init();

const app = createApp(App);
const pinia = createPinia();

app.use(pinia);
app.use(router);

app.mount("#app");