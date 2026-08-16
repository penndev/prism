<template>
  <a-card class="proxy-panel" :title="t('proxy.title')">
    <!-- ĺ˝ĺéä¸­çäťŁçćĺĄĺ¨ -->
    <div class="proxy-current-server">
      <span class="proxy-label">{{ t("proxy.currentServerLabel") }}</span>
      <span class="proxy-value">
        {{
          serverStore.selectedServer?.remark ||
          serverStore.selectedServer?.host ||
          t("proxy.noSelectedServer")
        }}
      </span>
      <a-button
        v-if="serverStore.selectedServer"
        type="link"
        size="small"
        danger
        @click="serverStore.selectedServer = null"
      >
        {{ t("proxy.removeButton") }}
      </a-button>
    </div>

    <!-- ćŞéćĺĄĺ¨ćśçćç¤ş -->
    <div v-if="!serverStore.selectedServer" class="proxy-mode-tip">
      {{ t("proxy.selectTip") }}
    </div>

    <!-- äťŁçć¨Ąĺźďźćĺ¨ / TUN -->
    <a-radio-group v-model:value="proxyMode" class="proxy-mode-group">
      <a-radio-button value="manual">
        {{ t("proxy.mode.manual") }}
      </a-radio-button>
      <a-radio-button value="tun">
        {{ t("proxy.mode.tun") }}
      </a-radio-button>
    </a-radio-group>

    <!-- PAC / č§ĺďźçźčžĺ¨ĺĽĺŁ -->
    <div class="proxy-pac-section">
      <div class="proxy-pac-line">
        <span class="proxy-pac-tag">{{ t("settings.pacTitle") }}</span>
        <span class="proxy-pac-dot" aria-hidden="true"></span>
        <a
          href="#"
          class="proxy-pac-link proxy-pac-js"
          :class="{ 'is-disabled': !webBaseURL }"
          @click.prevent="openPacEditor"
        >
          {{ t("settings.pacOpenEditor") }}
        </a>
        <span class="proxy-pac-dot" aria-hidden="true"></span>
        <a
          href="#"
          class="proxy-pac-link proxy-pac-js"
          :class="{ 'is-disabled': !pacScriptURL }"
          :title="pacScriptURL || undefined"
          @click.prevent="copyPacScriptURL"
        >
          {{ t("settings.pacScriptUrl") }}
        </a>
      </div>
      <div class="proxy-pac-line">
        <span class="proxy-pac-tag">{{ t("settings.ruleTitle") }}</span>
        <span class="proxy-pac-dot" aria-hidden="true"></span>
        <a
          href="#"
          class="proxy-pac-link proxy-pac-js"
          :class="{ 'is-disabled': !webBaseURL }"
          @click.prevent="openRuleEditor"
        >
          {{ ruleModeText }}
        </a>
        <template v-if="ruleAreaNamesText">
          <span class="proxy-pac-dot" aria-hidden="true"></span>
          <a
            href="#"
            class="proxy-pac-link proxy-pac-js proxy-rule-areas"
            :class="{ 'is-disabled': !webBaseURL }"
            :title="ruleAreaNamesText"
            @click.prevent="openRuleEditor"
          >
            {{ ruleAreaNamesText }}
          </a>
        </template>
      </div>
    </div>
  </a-card>
</template>

<script setup>
import { ref, watch, computed, onMounted, onUnmounted } from "vue";
import { theme, message } from "ant-design-vue";
import { Events } from "@wailsio/runtime";

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
import {
  AppConfig,
  OpenExternalURL,
} from "@bindings/desktop/internal/appconst";

const { token } = theme.useToken();
const settingsStore = useSettingsStore();
const serverStore = useServerStore();

const proxyMode = ref("manual");
const ruleMode = ref("global"); // global | match | bypass
const ruleAreaNames = ref([]);
let ruleEventOff = null;

/** ćŹĺ°äťŁç Web ćĺĄć šĺ°ĺďźç¨äşćĺź PAC / č§ĺçźčžĺ¨ */
const webBaseURL = computed(() => {
  const rawHost = (settingsStore.proxy.host || "").trim();
  const host =
    rawHost === "0.0.0.0" || rawHost === "" ? "127.0.0.1" : rawHost;
  const port = Number(settingsStore.proxy.port);

  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    return "";
  }

  return `http://${host}:${port}`;
});

/** PAC čćŹĺŻšĺ¤ URLďźäžćľč§ĺ¨/çłťçťĺźç¨ďź */
const pacScriptURL = computed(() =>
  webBaseURL.value ? `${webBaseURL.value}/pac.js` : "",
);

const ruleModeText = computed(() => {
  switch (ruleMode.value) {
    case "match":
      return t("settings.ruleStatus.match");
    case "bypass":
      return t("settings.ruleStatus.bypass");
    default:
      return t("settings.ruleStatus.global");
  }
});

const ruleAreaNamesText = computed(() =>
  (ruleAreaNames.value || []).filter(Boolean).join(", "),
);

function deriveRuleStatus(cfg) {
  switch (cfg?.mode) {
    case "proxy":
      return "match";
    case "bypass":
      return "bypass";
    default:
      return "global";
  }
}

async function loadRuleStatus() {
  try {
    const status = await Storage.GetRuleStatus();
    ruleMode.value = deriveRuleStatus(status);
    ruleAreaNames.value = Array.isArray(status?.areaNames)
      ? status.areaNames
      : [];
  } catch {
    ruleMode.value = "global";
    ruleAreaNames.value = [];
  }
}

/** ĺ¨ćľč§ĺ¨ä¸­ćĺź PAC çźčžĺ¨éĄľé˘ */
async function openPacEditor() {
  if (!webBaseURL.value) {
    message.warning(t("settings.pacNeedPort"));
    return;
  }

  try {
    await OpenExternalURL(`${webBaseURL.value}/pac/`);
  } catch (e) {
    message.error(e?.message || t("settings.pacOpenFailed"));
  }
}

/** ĺ¨ćľč§ĺ¨ä¸­ćĺźĺ°ĺč§ĺçźčžĺ¨ */
async function openRuleEditor() {
  if (!webBaseURL.value) {
    message.warning(t("settings.pacNeedPort"));
    return;
  }

  try {
    await OpenExternalURL(`${webBaseURL.value}/rule/`);
  } catch (e) {
    message.error(e?.message || t("settings.pacOpenFailed"));
  }
}

/** ĺ¤ĺś PAC čćŹ URL ĺ°ĺŞč´´ćż */
async function copyPacScriptURL() {
  if (!pacScriptURL.value) {
    message.warning(t("settings.pacNeedPort"));
    return;
  }

  try {
    await navigator.clipboard.writeText(pacScriptURL.value);
    message.success(t("settings.pacCopied"));
  } catch {
    message.error(t("settings.pacCopyFailed"));
  }
}

onMounted(async () => {
  await loadRuleStatus();
  try {
    const appConst = await AppConfig();
    ruleEventOff = Events.On(appConst.EventNameRuleChanged, () => {
      loadRuleStatus();
    });
  } catch {
    // ignore
  }
});

onUnmounted(() => {
  if (typeof ruleEventOff === "function") {
    ruleEventOff();
    ruleEventOff = null;
  }
});

/** ĺć˘äťŁçć¨ĄĺźćśéçĽĺçŤŻ SetMode */
watch(
  proxyMode,
  async (mode, prevMode) => {
    const revert = prevMode ?? "manual";
    try {
      await SetMode(mode);
    } catch (e) {
      if (mode !== revert) {
        proxyMode.value = revert;
      }
      message.error(e?.message ?? "");
    }
  },
  { immediate: true },
);

/** éä¸­/ĺćśćĺĄĺ¨ćśďźćäšĺéćŠĺšśčŽžç˝Ž/ĺć­˘čżç¨äťŁç */
watch(
  () => serverStore.selectedServer,
  async (server) => {
    try {
      if (server?.host && server?.protocol) {
        await Storage.SetSelectedServer(server);
        await SetRemote(extendServerItem(server)._id);
      } else {
        // ć¸çéä¸­ćĺĄĺ¨ďźä¸ćŹĄĺŻĺ¨ä¸čŞĺ¨çťĺ˝äşă
        await Storage.ClearSelectedServer();
        // await SetStop(); webćĺĄéčŚä¸ç´éŠťçă
      }
    } catch (e) {
      message.error(e?.message || t("serverList.operationFailed"));
    }
  },
  { immediate: true },
);

/** ćŹĺ°äťŁççĺŹĺ°ĺĺć´ćśéĺŻ SetStart */
watch(
  () => settingsStore.proxy,
  debounce(async () => {
    const { host, port, username, password } = settingsStore.proxy;
    try {
      await SetStart(`${host}:${port}`, username, password);
    } catch (e) {
      message.error(e?.message || t("serverList.operationFailed"));
    }
  }, 1000),
  { deep: true, immediate: true },
);
</script>

<style lang="scss" scoped>
/* äťŁçć§ĺśĺĄç */
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

  /* ĺ˝ĺćĺĄĺ¨čĄ */
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

  /* ćŞéćĺĄĺ¨ćç¤ş */
  .proxy-mode-tip {
    font-size: 13px;
    color: v-bind("token.colorTextSecondary");
    padding: 4px 0;
  }

  /* ć¨Ąĺźĺć˘ćéŽçť */
  .proxy-mode-group {
    width: 100%;
    margin-bottom: 6px;
    display: flex;

    :deep(.ant-radio-button-wrapper) {
      flex: 1;
      text-align: center;
    }
  }

  /* PAC / č§ĺéžćĽĺş */
  .proxy-pac-section {
    margin-top: 2px;
    padding-top: 6px;
    border-top: 1px solid v-bind("token.colorBorderSecondary");
  }

  .proxy-pac-line {
    font-size: 12px;
    line-height: 1.5;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0 2px;
  }

  .proxy-pac-line + .proxy-pac-line {
    margin-top: 4px;
  }

  .proxy-pac-tag {
    color: v-bind("token.colorTextSecondary");
  }

  .proxy-pac-dot {
    color: v-bind("token.colorTextQuaternary");
    user-select: none;
    padding: 0 3px;

    &::before {
      content: "\00b7";
    }
  }

  a.proxy-pac-link.proxy-rule-areas {
    flex: 1 1 80px;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .proxy-pac-link {
    color: v-bind("token.colorText");
    cursor: pointer;
    text-decoration: none;

    &:hover {
      color: #1677ff;
      text-decoration: underline;
    }

    &.is-disabled {
      color: v-bind("token.colorTextDisabled");
      cursor: not-allowed;
      pointer-events: none;
    }
  }

  a.proxy-pac-link.proxy-pac-js {
    color: #1677ff;
    font-weight: 500;

    &:hover {
      text-decoration: underline;
    }

    &.is-disabled {
      color: v-bind("token.colorTextDisabled");
    }
  }
}
</style>
