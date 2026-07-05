<template>
  <a-config-provider :theme="antdThemeConfig">
    <!-- 应用就绪后再渲染，避免 store 未初始化时闪烁 -->
    <div v-if="appReady" class="layout">
      <!-- 主内容区：左侧应用面板 + 右侧设置面板 -->
      <div class="main">
        <div class="app" :style="mainStyle">
          <!-- 标题栏：应用名 + 设置面板开关 -->
          <header class="app-hd">
            <span class="app-title">{{ t("app.title") }}</span>
            <a-switch v-model:checked="extensionVisible" size="small" />
          </header>

          <action-panel />
          <serve-panel />
        </div>

        <!-- 右侧设置扩展面板（可拖拽调整主栏宽度） -->
        <setting-panel v-if="extensionVisible" :start-width-resize="onResizeDivider" />
      </div>

      <!-- 底部日志栏（可在系统设置中开关） -->
      <div v-if="settingsStore.system.enableLogRecording" class="bottom">
        <bottom-bar />
      </div>
    </div>
  </a-config-provider>
</template>

<script setup>
import {
  ref,
  computed,
  watch,
  onBeforeMount,
  onMounted,
  onBeforeUnmount,
} from "vue";
import { theme } from "ant-design-vue";
import { Window } from "@wailsio/runtime";

import { useSettingsStore } from "@/stores/settings";
import { useServerStore } from "@/stores/server";
import { t } from "@/locale";
import { startAxisResize } from "@/utils";

import ActionPanel from "./components/ActionPanel.vue";
import ServePanel from "./components/ServePanel.vue";
import SettingPanel from "./components/SettingPanel.vue";
import BottomBar from "./components/BottomBar.vue";

// ---------------------------------------------------------------------------
// Store 与 Ant Design 主题 token
// ---------------------------------------------------------------------------

const settingsStore = useSettingsStore();
const serverStore = useServerStore();
const { token } = theme.useToken();

/** 跟随系统深浅色偏好（settings 为 auto 时生效） */
const colorScheme = window.matchMedia("(prefers-color-scheme: dark)");
const prefersColor = ref(colorScheme.matches);

/** Ant Design 全局主题：深浅色算法 + 按钮去阴影 */
const antdThemeConfig = computed(() => ({
  algorithm: [
    settingsStore.system.themeMode === "dark" || prefersColor.value
      ? theme.darkAlgorithm
      : theme.defaultAlgorithm,
  ],
  components: { Button: { primaryShadow: "none" } },
}));

// ---------------------------------------------------------------------------
// 双栏布局：主栏 ↔ 设置栏宽度规则（单位 px）
// ---------------------------------------------------------------------------
const MAIN_MIN = 400;
const MAIN_MAX = 600;
const RIGHT_MIN = 280;

/** 客户区宽度低于此值时自动折叠为单栏 */
/** SetSize 多为窗口外框宽度，略大于 SPLIT_MIN_INNER，尽量保证客户区够双栏 */
const SPLIT_MIN_INNER = 800;
const SPLIT_WINDOW_OUTER_WIDTH = SPLIT_MIN_INNER + 40;

/** 设置面板是否可见（可由标题栏开关或窗口过窄时自动关闭） */
/** 主栏当前宽度 */
/** store 初始化完成后才渲染 UI */
const extensionVisible = ref(true);
const mainWidth = ref(MAIN_MIN);
const appReady = ref(false);

/** 主栏 inline 样式：双栏固定宽度，单栏占满剩余空间 */
const mainStyle = computed(() =>
  extensionVisible.value
    ? {
      flex: `0 0 ${mainWidth.value}px`,
      width: `${mainWidth.value}px`,
      minWidth: 0,
    }
    : { flex: "1 1 auto", width: "100%", minWidth: 0 },
);

/** 根据窗口宽度同步双栏/单栏状态，并 clamp 主栏宽度 */
function applyLayoutToWindow() {
  const inner = window.innerWidth;

  if (inner < SPLIT_MIN_INNER) {
    extensionVisible.value = false;
    return;
  }

  extensionVisible.value = true;
  const cap = Math.min(MAIN_MAX, Math.max(MAIN_MIN, inner - RIGHT_MIN));
  if (mainWidth.value > cap) {
    mainWidth.value = cap;
  }
}

/** 拖拽分隔条时调整主栏宽度 */
function onResizeDivider(e) {
  startAxisResize(e, {
    axis: "x",
    startValue: mainWidth.value,
    min: MAIN_MIN,
    max: Math.min(MAIN_MAX, Math.max(MAIN_MIN, window.innerWidth - RIGHT_MIN)),
    onChange: (v) => {
      mainWidth.value = v;
    },
  });
}

/** 用户打开设置面板但窗口过窄时，自动扩窗至双栏最小宽度 */
watch(extensionVisible, async (visible) => {
  mainWidth.value = MAIN_MIN;

  if (!visible || window.innerWidth >= SPLIT_MIN_INNER) {
    return;
  }

  try {
    const { height } = await Window.Size();
    await Window.SetSize(SPLIT_WINDOW_OUTER_WIDTH, height);
  } finally {
    applyLayoutToWindow();
  }
});

onBeforeMount(async () => {
  await settingsStore.init();
  await serverStore.init();
  appReady.value = true;
});

onMounted(async () => {
  colorScheme.addEventListener("change", function (e) {
    prefersColor.value = e.matches;
  });
  applyLayoutToWindow();
  window.addEventListener("resize", applyLayoutToWindow);
});

onBeforeUnmount(() => {
  colorScheme.removeEventListener("change", function (e) {
    prefersColor.value = e.matches;
  });
  window.removeEventListener("resize", applyLayoutToWindow);
});
</script>

<style lang="scss" scoped>
/* 根布局：纵向 flex，主区 + 可选底部日志栏 */
.layout {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: v-bind("token.colorBgLayout");

  /* 主区：横向 flex，左 app + 右 setting */
  .main {
    flex: 1;
    display: flex;
    min-height: 0;
  }

  /* 左侧应用面板 */
  .app {
    box-sizing: border-box;
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 0 10px 8px;
    overflow: hidden;
    font-size: 14px;
    color: v-bind("token.colorText");
    background: v-bind("token.colorBgContainer");

    .app-hd {
      flex-shrink: 0;
      height: 48px;
      margin: 0 -10px;
      padding: 0 12px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      background: v-bind("token.colorBgElevated");
      border-bottom: 1px solid v-bind("token.colorBorderSecondary");

      .app-title {
        font-size: 16px;
        font-weight: 600;
        color: v-bind("token.colorText");
      }
    }
  }

  /* 底部日志栏 */
  .bottom {
    flex-shrink: 0;
    border-top: 1px solid v-bind("token.colorBorderSecondary");
    background: v-bind("token.colorBgElevated");
  }
}
</style>
