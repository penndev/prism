<template>
  <div class="bottom-area">
    <StatusPanel
      v-show="activePanel === PANEL_NAMES.STATUS"
      :panelHeightPx="panelHeightPx"
      :onClose="closePanel"
      :startResize="startResize"
    />
    <LogPanel
      v-show="activePanel === PANEL_NAMES.LOG"
      :panelHeightPx="panelHeightPx"
      :onClose="closePanel"
      :startResize="startResize"
    />
    <!-- 底部状态栏：状态日志 | 连接日志 -->
    <div class="status-bar">
      <span
        class="status-item"
        :class="{ active: activePanel === PANEL_NAMES.STATUS }"
        @click="togglePanel(PANEL_NAMES.STATUS)"
      >
        {{ t("log.statusTitle") }}
      </span>
      <span
        class="status-item"
        :class="{ active: activePanel === PANEL_NAMES.LOG }"
        @click="togglePanel(PANEL_NAMES.LOG)"
      >
        {{ t("log.connectionTitle") }}
      </span>
      <span class="status-spacer" />
      <span
        class="traffic-item"
        :title="`${t('log.trafficTotalDown')}: ${writeTotalText}`"
      >
        <ArrowDownOutlined />
        {{ writeSpeedText }}
      </span>
      <span
        class="traffic-item"
        :title="`${t('log.trafficTotalUp')}: ${readTotalText}`"
      >
        <ArrowUpOutlined />
        {{ readSpeedText }}
      </span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from "vue";
import { theme } from "ant-design-vue";
import { ArrowDownOutlined, ArrowUpOutlined } from "@ant-design/icons-vue";
import { t } from "@/locale";
import StatusPanel from "./bottombar/StatusPanel.vue";
import LogPanel from "./bottombar/LogPanel.vue";
import { TrafficBytes } from "@bindings/desktop/internal/proxy/proxy";
import { startAxisResize } from "@/utils";

const PANEL_HEIGHT_MIN = 80;
const PANEL_HEIGHT_MAX = 480;
const PANEL_NAMES = {
  STATUS: "status",
  LOG: "log",
};

const activePanel = ref(null);
const panelHeight = ref(160);

const { token } = theme.useToken();

const panelHeightPx = computed(() => `${panelHeight.value}px`);


function startResize(e) {
  startAxisResize(e, {
    axis: "y",
    startValue: panelHeight.value,
    min: PANEL_HEIGHT_MIN,
    max: PANEL_HEIGHT_MAX,
    onChange: (v) => {
      panelHeight.value = v;
    },
  });
}

function togglePanel(name) {
  activePanel.value = activePanel.value === name ? null : name;
}

function closePanel() {
  activePanel.value = null;
}


const readSpeedText = ref("0 B/s");
const writeSpeedText = ref("0 B/s");
const readTotalText = ref("0 B");
const writeTotalText = ref("0 B");
let trafficTimer = null;
let lastReadBytes = 0;
let lastWriteBytes = 0;
let lastSampleAt = 0;


function formatBytes(value) {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let next = value;
  let idx = 0;
  while (next >= 1024 && idx < units.length - 1) {
    next /= 1024;
    idx += 1;
  }
  const precision = idx === 0 ? 0 : next >= 100 ? 0 : next >= 10 ? 1 : 2;
  return `${next.toFixed(precision)} ${units[idx]}`;
}

async function sampleTraffic() {
  try {
    const [readBytes, writeBytes] = await TrafficBytes();
    const now = Date.now();
    if (lastSampleAt > 0) {
      const elapsedSeconds = (now - lastSampleAt) / 1000;
      if (elapsedSeconds > 0) {
        const readDelta = Math.max(0, readBytes - lastReadBytes);
        const writeDelta = Math.max(0, writeBytes - lastWriteBytes);
        readSpeedText.value = `${formatBytes(readDelta / elapsedSeconds)}/s`;
        writeSpeedText.value = `${formatBytes(writeDelta / elapsedSeconds)}/s`;
      }
    }
    readTotalText.value = formatBytes(readBytes);
    writeTotalText.value = formatBytes(writeBytes);
    lastReadBytes = readBytes;
    lastWriteBytes = writeBytes;
    lastSampleAt = now;
  } catch (e) {
    console.error("[BottomBar] TrafficBytes() failed:", e);
  }
}

onMounted(async () => {
  trafficTimer = window.setInterval(sampleTraffic, 1000);
});

onBeforeUnmount(() => {
  if (trafficTimer !== null) {
    window.clearInterval(trafficTimer);
    trafficTimer = null;
  }
});
</script>

<style scoped lang="scss">
.bottom-area {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.status-bar {
  height: 28px;
  padding: 0 10px;
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: v-bind("token.colorTextSecondary");
  background: v-bind("token.colorBgContainer");
  border-top: 1px solid v-bind("token.colorBorderSecondary");

  .status-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    cursor: pointer;
    user-select: none;

    &:hover {
      color: #1677ff;
    }

    &.active {
      color: #1677ff;
      font-weight: 500;
    }
  }

  .traffic-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    user-select: none;
    color: v-bind("token.colorTextSecondary");
  }

  .status-spacer {
    flex: 1;
  }
}
</style>
