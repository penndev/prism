<template>
  <a-config-provider :theme="antdThemeConfig">
    <div v-if="appReady" class="layout">
      <div class="main">
        <div class="app" :style="mainStyle">
          <header class="app-hd">
            <span class="app-title">{{ t("app.title") }}</span>
            <a-switch v-model:checked="extensionVisible" size="small" />
          </header>
          <action-panel />
          <serve-panel />
        </div>
        <setting-panel
          v-if="extensionVisible"
          :start-width-resize="onResizeDivider"
        />
      </div>
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
import ActionPanel from "./components/ActionPanel.vue";
import ServePanel from "./components/ServePanel.vue";
import SettingPanel from "./components/SettingPanel.vue";
import BottomBar from "./components/BottomBar.vue";
import { startAxisResize } from "@/utils";

const settingsStore = useSettingsStore();
const serverStore = useServerStore();
const { token } = theme.useToken();
const colorScheme = window.matchMedia("(prefers-color-scheme: dark)");
const prefersColor = ref(colorScheme.matches);
const antdThemeConfig = computed(() => ({
  algorithm: [
    settingsStore.system.themeMode === "dark" || prefersColor.value
      ? theme.darkAlgorithm
      : theme.defaultAlgorithm,
  ],
  components: { Button: { primaryShadow: "none" } },
}));

// —— 主栏 ↔ 设置栏：宽度规则（单位 px）——
const MAIN_MIN = 400;
const MAIN_MAX = 600;
const RIGHT_MIN = 280;
const SPLIT_MIN_INNER = 800;
/** SetSize 多为窗口外框宽度，略大于 SPLIT_MIN_INNER，尽量保证客户区够双栏 */
const SPLIT_WINDOW_OUTER_WIDTH = SPLIT_MIN_INNER + 40;

const extensionVisible = ref(true);
const mainWidth = ref(MAIN_MIN);
const appReady = ref(false);

const mainStyle = computed(() =>
  extensionVisible.value
    ? { flex: `0 0 ${mainWidth.value}px`, width: `${mainWidth.value}px`, minWidth: 0 }
    : { flex: "1 1 auto", width: "100%", minWidth: 0 },
);

/** 双栏时：当前窗口下主栏允许的最大宽度（保证右侧至少 RIGHT_MIN） */
function mainMaxFor(innerW) {
  return Math.min(MAIN_MAX, Math.max(MAIN_MIN, innerW - RIGHT_MIN));
}

function applyLayoutToWindow() {
  const inner = window.innerWidth;
  if (inner < SPLIT_MIN_INNER) {
    extensionVisible.value = false;
    return;
  }
  extensionVisible.value = true;
  const cap = mainMaxFor(inner);
  if (mainWidth.value > cap) mainWidth.value = cap;
}

function onResizeDivider(e) {
  startAxisResize(e, {
    axis: "x",
    startValue: mainWidth.value,
    min: MAIN_MIN,
    max: mainMaxFor(window.innerWidth),
    onChange: (v) => {
      mainWidth.value = v;
    },
  });
}

watch(extensionVisible, async (visible) => {
  mainWidth.value = MAIN_MIN;
  if (!visible || window.innerWidth >= SPLIT_MIN_INNER) return;
  try {
    const { height } = await Window.Size();
    await Window.SetSize(SPLIT_WINDOW_OUTER_WIDTH, height);
  } finally {
    applyLayoutToWindow();
  }
});

function onSystemColorChange(e) {
  prefersColor.value = e.matches;
}

onBeforeMount(async () => {
  await settingsStore.init();
  await serverStore.init();
  appReady.value = true;
});

onMounted(async () => {
  colorScheme.addEventListener("change", onSystemColorChange);
  applyLayoutToWindow();
  window.addEventListener("resize", applyLayoutToWindow);
});

onBeforeUnmount(() => {
  colorScheme.removeEventListener("change", onSystemColorChange);
  window.removeEventListener("resize", applyLayoutToWindow);
});
</script>

<style lang="scss" scoped>
.layout {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: v-bind("token.colorBgLayout");

  .main {
    flex: 1;
    display: flex;
    min-height: 0;
  }

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

  .bottom {
    flex-shrink: 0;
    border-top: 1px solid v-bind("token.colorBorderSecondary");
    background: v-bind("token.colorBgElevated");
  }
}
</style>
