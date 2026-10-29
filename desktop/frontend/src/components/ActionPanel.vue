<template>
  <a-card class="proxy-panel" :title="t('proxy.title')">
    <template #extra>
      <a-tooltip :title="t('settings.ruleTitle')">
        <a-button
          type="text"
          size="small"
          class="card-extra-btn"
          :disabled="!settingsStore.webBaseURL"
          @click="openRuleEditor"
        >
          <svg class="head-icon" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M19.14,12.94c0.04-0.3,0.06-0.61,0.06-0.94c0-0.32-0.02-0.64-0.07-0.94l2.03-1.58c0.18-0.14,0.23-0.41,0.12-0.61l-1.92-3.32c-0.12-0.22-0.37-0.29-0.59-0.22l-2.39,0.96c-0.5-0.38-1.03-0.7-1.62-0.94L14.4,2.81c-0.04-0.24-0.24-0.41-0.48-0.41h-3.84c-0.24,0-0.43,0.17-0.47,0.41L9.25,5.35C8.66,5.59,8.12,5.92,7.63,6.29L5.24,5.33c-0.22-0.08-0.47,0-0.59,0.22L2.74,8.87C2.62,9.08,2.66,9.34,2.86,9.48l2.03,1.58C4.84,11.36,4.8,11.69,4.8,12s0.02,0.64,0.07,0.94l-2.03,1.58c-0.18,0.14-0.23,0.41-0.12,0.61l1.92,3.32c0.12,0.22,0.37,0.29,0.59,0.22l2.39-0.96c0.5,0.38,1.03,0.7,1.62,0.94l0.36,2.54c0.05,0.24,0.24,0.41,0.48,0.41h3.84c0.24,0,0.44-0.17,0.47-0.41l0.36-2.54c0.59-0.24,1.13-0.56,1.62-0.94l2.39,0.96c0.22,0.08,0.47,0,0.59-0.22l1.92-3.32c0.12-0.22,0.07-0.47-0.12-0.61L19.14,12.94z M12,15.6c-1.98,0-3.6-1.62-3.6-3.6s1.62-3.6,3.6-3.6s3.6,1.62,3.6,3.6S13.98,15.6,12,15.6z"
            />
          </svg>
        </a-button>
      </a-tooltip>
    </template>

    <div v-if="!serverStore.selectedServer" class="proxy-empty">
      <span class="proxy-empty-text">{{ t("proxy.selectTip") }}</span>
    </div>

    <template v-else>
      <div class="proxy-current-server">
        <span class="proxy-label">{{ t("proxy.currentServerLabel") }}</span>
        <span class="proxy-value">
          {{
            serverStore.selectedServer.remark ||
            serverStore.selectedServer.host
          }}
        </span>
        <a-button
          type="link"
          size="small"
          danger
          @click="serverStore.selectedServer = null"
        >
          {{ t("proxy.removeButton") }}
        </a-button>
      </div>

      <a-radio-group v-model:value="proxyMode" class="proxy-mode-group">
        <a-radio-button value="manual">
          {{ t("proxy.mode.manual") }}
        </a-radio-button>
        <a-radio-button value="tun">
          {{ t("proxy.mode.tun") }}
        </a-radio-button>
      </a-radio-group>
    </template>
  </a-card>
</template>

<script setup>
import { ref, watch } from "vue";
import { theme, message } from "ant-design-vue";

import { useServerStore } from "../stores/server";
import { useSettingsStore } from "@/stores/settings";
import { t } from "@/locale";
import { extendServerItem, debounce } from "@/utils";

import { Storage } from "@bindings/desktop/internal/storage";
import {
  SetStart,
  SetRemote,
  SetMode,
} from "@bindings/desktop/internal/proxy/proxy";
import { OpenExternalURL } from "@bindings/desktop/internal/appconst";

const { token } = theme.useToken();
const settingsStore = useSettingsStore();
const serverStore = useServerStore();

const proxyMode = ref("manual");

/** 在浏览器中打开规则编辑器 */
async function openRuleEditor() {
  if (!settingsStore.webBaseURL) {
    message.warning(t("settings.pacNeedPort"));
    return;
  }

  try {
    await OpenExternalURL(`${settingsStore.webBaseURL}/rule/`);
  } catch (e) {
    message.error(e?.message || t("settings.pacOpenFailed"));
  }
}

/** 切换代理模式时通知后端 SetMode 并持久化 */
watch(proxyMode, async (mode, prevMode) => {
  const revert = prevMode ?? "manual";
  try {
    await SetMode(mode);
    settingsStore.proxyMode = mode;
  } catch (e) {
    if (mode !== revert) {
      proxyMode.value = revert;
    }
    message.error(e?.message ?? "");
  }
});

/** 选中/取消服务器时：持久化选择并设置/停止远程代理 */
async function applySelectedServer(server) {
  try {
    if (server?.host && server?.protocol) {
      await Storage.SetSelectedServer(server);
      await SetRemote(extendServerItem(server)._id);
    } else {
      // 本地监听要一直开着给 web 管理页用，只卸掉远程节点。
      await Storage.ClearSelectedServer();
      await SetRemote("");
      proxyMode.value = "manual";
    }
  } catch (e) {
    message.error(e?.message || t("serverList.operationFailed"));
  }
}
watch(() => serverStore.selectedServer, applySelectedServer);

/** 本地代理监听地址变更时重启 SetStart */
async function startLocalProxy() {
  const { host, port, username, password } = settingsStore.proxy;
  try {
    await SetStart(`${host}:${port}`, username, password);
  } catch (e) {
    message.error(e?.message || t("serverList.operationFailed"));
  }
}
watch(() => settingsStore.proxy, debounce(startLocalProxy, 1000), { deep: true });

// 启动顺序必须是 本地监听 -> 远程节点 -> 恢复代理模式：
// 后端 startTunDev 要求远程节点已就绪，否则恢复 tun 模式会直接被拒。
(async () => {
  await startLocalProxy();
  await applySelectedServer(serverStore.selectedServer);
  proxyMode.value = settingsStore.proxyMode || "manual";
})();
</script>

<style lang="scss" scoped>
/* 代理控制卡片 */
.proxy-panel {
  flex-shrink: 0;
  border-radius: 10px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

  :deep(.ant-card-head) {
    min-height: auto;
    padding: 0 10px;
  }

  :deep(.ant-card-head-title) {
    padding: 10px 0;
    font-size: 14px;
  }

  :deep(.ant-card-body) {
    padding: 12px;
  }

  /* 当前服务器行 */
  .proxy-current-server {
    margin-bottom: 6px;
    padding: 3px 6px;
    background: v-bind("token.colorFillAlter");
    border-radius: 8px;
    display: flex;
    align-items: center;
    flex-wrap: nowrap;
    gap: 6px;

    .proxy-label {
      font-size: 12px;
      color: v-bind("token.colorTextSecondary");
      margin-right: 0;
    }

    .proxy-value {
      font-size: 13px;
      font-weight: 500;
      color: #1677ff;
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .proxy-empty {
    display: flex;
    align-items: center;
    min-height: 56px;
    padding: 10px 12px;
    background: v-bind("token.colorFillAlter");
    border: 1px dashed v-bind("token.colorBorder");
    border-radius: 8px;

    .proxy-empty-text {
      font-size: 13px;
      line-height: 1.45;
      color: v-bind("token.colorTextSecondary");
    }
  }

  /* 模式切换按钮组 */
  .proxy-mode-group {
    width: 100%;
    margin-bottom: 0;
    display: flex;

    :deep(.ant-radio-button-wrapper) {
      flex: 1;
      text-align: center;
    }
  }

  .card-extra-btn {
    width: 28px;
    height: 28px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;

    .head-icon {
      width: 18px;
      height: 18px;
      display: block;
    }
  }
}
</style>
