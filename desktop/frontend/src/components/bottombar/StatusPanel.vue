<template>
  <div class="panel" :style="{ height: panelHeightPx }">
    <div
      class="panel-resize-handle"
      :title="t('log.dragToResize')"
      @mousedown="startResize($event)"
    />
    <div class="panel-header">
      <span>{{ t("log.statusTitle") }}</span>
      <div class="panel-actions">
        <a-button type="text" size="small" @click="clearStatus">
          {{ t("log.clear") }}
        </a-button>
        <a-button type="text" size="small" @click="onClose">
          <CloseOutlined />
        </a-button>
      </div>
    </div>
    <pre class="panel-content">{{ displayText }}</pre>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from "vue";
import { Events } from "@wailsio/runtime";
import { CloseOutlined } from "@ant-design/icons-vue";
import { t } from "@/locale";
import { theme } from "ant-design-vue";
import { AppConfig } from "@bindings/desktop/internal/appconst";

const { token } = theme.useToken();

defineProps({
  panelHeightPx: String,
  onClose: Function,
  startResize: Function,
});

const MAX_STATUS_LINES = 1000;

const lines = ref([]);
const displayText = computed(
  () => lines.value.join("\n") || t("log.statusEmpty"),
);

function clearStatus() {
  lines.value = [];
}

let statusEventOff = null;
onMounted(async () => {
  try {
    const appConst = await AppConfig();
    statusEventOff = Events.On(appConst.LogTypeName_STATUS, (eventPayload) => {
      lines.value.push(String(eventPayload.data));
      if (lines.value.length > MAX_STATUS_LINES) {
        lines.value.shift();
      }
    });
  } catch (e) {
    console.error("[StatusPanel] AppConfig() failed:", e);
  }
});

onBeforeUnmount(() => {
  if (typeof statusEventOff === "function") {
    statusEventOff();
  }
});
</script>

<style scoped lang="scss">
.panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding-top: 6px;
  border-top: 1px solid v-bind("token.colorBorderSecondary");
  background: v-bind("token.colorBgContainer");
  position: relative;
  box-sizing: border-box;

  .panel-resize-handle {
    position: absolute;
    left: 0;
    right: 0;
    top: 0;
    height: 4px;
    cursor: ns-resize;
    z-index: 1;
    background: transparent;
    transition: background 0.15s;
    &:hover {
      background: rgba(66, 133, 244, 0.2);
    }
  }

  .panel-header {
    flex-shrink: 0;
    height: 32px;
    padding: 0 10px;
    display: flex;
    align-items: center;
    justify-content: space-between;

    .panel-actions {
      display: flex;
      align-items: center;
      gap: 4px;
    }
    font-size: 12px;
    font-weight: 500;
    color: v-bind("token.colorText");
    border-bottom: 1px solid v-bind("token.colorBorderSecondary");
    background: v-bind("token.colorBgContainer");
  }

  .panel-content {
    flex: 1;
    min-height: 0;
    margin: 0;
    padding: 8px 10px;
    overflow: auto;
    font-family:
      ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono",
      "Courier New", monospace;
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
    color: v-bind("token.colorText");
    background: v-bind("token.colorFillAlter");
  }
}
</style>
