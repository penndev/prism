<template>
  <div class="socks5-extension">
    <div class="socks5-divider" @mousedown="startWidthResize" />
    <div class="socks5-extension-body">
      <div class="settings-panel">
        <section class="setting-section">
          <div class="section-header">
            <span class="section-title">{{ t("settings.localProxy") }}</span>
          </div>
          <a-form layout="vertical" class="section-form">
            <a-form-item :label="t('settings.ipAddress')">
              <a-select
                v-model:value="settingsStore.proxy.host"
                :placeholder="t('settings.ipPlaceholder')"
                style="width: 100%"
              >
                <a-select-option value="127.0.0.1">127.0.0.1</a-select-option>
                <a-select-option value="0.0.0.0">0.0.0.0</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item :label="t('settings.port')">
              <a-input-number
                v-model:value="settingsStore.proxy.port"
                :min="1"
                :max="65535"
                :placeholder="t('settings.portPlaceholder')"
                style="width: 100%"
              />
            </a-form-item>
            <a-form-item :label="t('settings.username')">
              <a-input
                v-model:value="settingsStore.proxy.username"
                :placeholder="t('settings.usernamePlaceholder')"
                allow-clear
              />
            </a-form-item>
            <a-form-item :label="t('settings.password')">
              <a-input-password
                v-model:value="settingsStore.proxy.password"
                :placeholder="t('settings.passwordPlaceholder')"
                allow-clear
              />
            </a-form-item>
          </a-form>
        </section>

        <section class="setting-section">
          <div class="section-header">
            <span class="section-title">{{ t("settings.latencyTest") }}</span>
          </div>
          <a-form layout="vertical" class="section-form">
            <a-form-item :label="t('settings.latencyTestHost')">
              <a-input
                v-model:value="settingsStore.latencyTest.host"
                :placeholder="t('settings.latencyTestHostPlaceholder')"
                allow-clear
              />
            </a-form-item>
            <a-form-item :label="t('settings.sortByLatencyAfterPing')">
              <a-switch
                v-model:checked="settingsStore.latencyTest.sortAfterPing"
              />
              <span class="setting-desc">
                {{ t("settings.sortByLatencyAfterPingDesc") }}
              </span>
            </a-form-item>
          </a-form>
        </section>

        <section class="setting-section">
          <div class="section-header">
            <span class="section-title">{{ t("settings.system") }}</span>
          </div>
          <a-form layout="vertical" class="section-form">
            <a-form-item :label="t('settings.systemLanguage')">
              <a-select
                v-model:value="settingsStore.system.language"
                :placeholder="t('settings.selectLanguage')"
                style="width: 100%"
              >
                <a-select-option value="zh-CN">简体中文</a-select-option>
                <a-select-option value="en">English</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item :label="t('settings.theme')">
              <a-select
                v-model:value="settingsStore.system.themeMode"
                style="width: 100%"
              >
                <a-select-option value="system">
                  {{ t("settings.theme.system") }}
                </a-select-option>
                <a-select-option value="light">
                  {{ t("settings.theme.light") }}
                </a-select-option>
                <a-select-option value="dark">
                  {{ t("settings.theme.dark") }}
                </a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item :label="t('settings.startupOnBoot')">
              <a-switch v-model:checked="settingsStore.system.startupOnBoot" />
              <span class="setting-desc">
                {{ t("settings.startupOnBootDesc") }}
              </span>
            </a-form-item>
            <a-form-item :label="t('settings.enableLogRecording')">
              <a-switch
                v-model:checked="settingsStore.system.enableLogRecording"
              />
              <span class="setting-desc">
                {{ t("settings.enableLogRecordingDesc") }}
              </span>
            </a-form-item>
          </a-form>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { PropType } from "vue";
import { useSettingsStore } from "@/stores/settings";
import { t } from "@/locale";
import { theme } from "ant-design-vue";

const { token } = theme.useToken();
const settingsStore = useSettingsStore();

defineProps({
  startWidthResize: {
    type: Function as PropType<(e: MouseEvent) => void>,
    required: true,
  },
});
</script>

<style lang="scss" scoped>
.socks5-extension {
  flex: 1;
  display: flex;
  background: v-bind("token.colorBgContainer");
  border-left: 1px solid v-bind("token.colorBorderSecondary");

  .socks5-divider {
    width: 4px;
    cursor: ew-resize;
    background: transparent;
    transition: background 0.15s;

    &:hover {
      background: rgba(66, 133, 244, 0.2);
    }
  }

  .socks5-extension-body {
    flex: 1;
    padding: 10px 12px;
    overflow-y: auto;
  }

  .settings-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .setting-section {
    padding: 12px 14px;
    border-radius: 10px;
    background: v-bind("token.colorFillQuaternary");
    border: 1px solid v-bind("token.colorBorderSecondary");
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  }

  .section-header {
    margin-bottom: 12px;
    padding-bottom: 10px;
    border-bottom: 1px solid v-bind("token.colorBorderSecondary");
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: v-bind("token.colorText");
    letter-spacing: 0.02em;
  }

  .section-form {
    :deep(.ant-form-item:last-child) {
      margin-bottom: 0;
    }
  }

  .setting-desc {
    display: block;
    margin-top: 6px;
    font-size: 12px;
    color: v-bind("token.colorTextSecondary");
    line-height: 1.45;
  }
}
</style>
