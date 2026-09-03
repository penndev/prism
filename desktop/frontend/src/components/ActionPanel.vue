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
          <FilterOutlined />
        </a-button>
      </a-tooltip>
    </template>

    <div v-if="!serverStore.selectedServer" class="proxy-empty">
      <span class="proxy-empty-icon">
        <CloudServerOutlined />
      </span>
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
import { CloudServerOutlined, FilterOutlined } from "@ant-design/icons-vue";

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
    gap: 10px;
    min-height: 56px;
    padding: 10px 12px;
    background: v-bind("token.colorFillAlter");
    border: 1px dashed v-bind("token.colorBorder");
    border-radius: 8px;

    .proxy-empty-icon {
      flex-shrink: 0;
      width: 32px;
      height: 32px;
      border-radius: 8px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      background: v-bind("token.colorPrimaryBg");
      color: v-bind("token.colorPrimary");
      font-size: 16px;
    }

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
    width: 24px;
    height: 24px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;

    .anticon {
      font-size: 14px;
    }
  }
}
</style>
