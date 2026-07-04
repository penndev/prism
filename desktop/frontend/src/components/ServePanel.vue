<template>
  <a-card class="server-panel" :title="t('serverList.title')">
    <template #extra>
      <div class="card-extra-actions">
        <a-tooltip :title="t('serverList.pingAll')">
          <a-button
            type="text"
            size="small"
            :loading="pingingAll"
            :disabled="servers.length === 0"
            @click="pingAllServers"
          >
            <ThunderboltOutlined />
          </a-button>
        </a-tooltip>
        <a-tooltip :title="t('serverList.sortByLatency')">
          <a-button
            type="text"
            size="small"
            :disabled="servers.length === 0"
            @click="sortByLatency"
          >
            <SortAscendingOutlined />
          </a-button>
        </a-tooltip>
        <a-tooltip :title="t('serverList.openSubscribeEditor')">
          <a-button type="text" size="small" @click="openSubscribeEditor">
            <EditOutlined />
          </a-button>
        </a-tooltip>
      </div>
    </template>
    <div class="list" v-if="servers.length > 0">
      <a-list :data-source="servers" row-key="__id" bordered>
        <template #renderItem="{ item }">
          <a-list-item :class="{ active: isServerSelected(item) }" @click="selectedServer = item">
            <template #actions>
              <a-button type="text" size="small" @click.stop="edit.open(item)">
                <EditOutlined />
              </a-button>
              <a-button
                type="text"
                danger
                size="small"
                @click.stop="deleteModal(item)"
              >
                <DeleteOutlined />
              </a-button>
            </template>

            <a-list-item-meta>
              <template #title>
                <span class="server-title-row">
                  <CheckCircleFilled v-if="isServerSelected(item)" class="selected-icon" />
                  <span class="server-host" :title="item.remark || item.host">
                    {{ item.remark || item.host }}
                  </span>
                  <span
                    v-if="latencyById[item.__id] !== undefined"
                    class="latency-inline"
                  >
                    <span
                      v-if="latencyById[item.__id] >= 0"
                      :class="getLatencyClass(latencyById[item.__id])"
                    >
                      {{ latencyById[item.__id] }}ms
                    </span>
                    <span v-else class="latency-error">{{
                      t("serverList.pingFailed")
                    }}</span>
                  </span>
                </span>
              </template>
              <template #description>
                <span
                  class="server-meta"
                  :title="`${item.protocol} | ${
                    item.username || t('serverList.noAuth')
                  }`"
                >
                  <span class="server-protocol">{{ item.protocol }}</span>
                  <span class="server-separator">|</span>
                  <span class="server-username">{{
                    item.username || t("serverList.noAuth")
                  }}</span>
                </span>
              </template>
            </a-list-item-meta>
          </a-list-item>
        </template>
      </a-list>
    </div>

    <div class="empty" v-else>
      <a-empty :description="t('serverList.emptyDescription')" />
    </div>

    <a-button type="primary" block class="add-server-btn" @click="edit.open()">
      <PlusOutlined />
      {{ t("serverList.addButton") }}
    </a-button>

    <a-modal
      v-model:open="edit.visible"
      :title="edit.title"
      :confirm-loading="edit.loading"
      @ok="edit.submit"
      @cancel="edit.visible = false"
    >
      <a-form
        ref="editRef"
        :model="edit.form"
        :rules="edit.rules"
        layout="vertical"
      >
        <a-form-item :label="t('serverList.hostLabel')" name="host">
          <a-input
            v-model:value="edit.form.host"
            :placeholder="t('serverList.hostPlaceholder')"
            allow-clear
          />
        </a-form-item>

        <a-form-item :label="t('serverList.remarkLabel')" name="remark">
          <a-input
            v-model:value="edit.form.remark"
            :placeholder="t('serverList.remarkPlaceholder')"
            allow-clear
          />
        </a-form-item>

        <a-form-item :label="t('serverList.protocolLabel')" name="protocol">
          <a-select
            v-model:value="edit.form.protocol"
            :placeholder="t('serverList.selectProtocol')"
          >
            <a-select-option
              v-for="scheme in proxySchemes"
              :key="scheme"
              :value="scheme"
              >{{ scheme }}</a-select-option
            >
          </a-select>
        </a-form-item>

        <a-form-item :label="t('serverList.usernameLabel')" name="username">
          <a-input
            v-model:value="edit.form.username"
            :placeholder="t('serverList.usernamePlaceholder')"
            allow-clear
          />
        </a-form-item>

        <a-form-item :label="t('serverList.passwordLabel')" name="password">
          <a-input-password
            v-model:value="edit.form.password"
            :placeholder="t('serverList.passwordPlaceholder')"
            allow-clear
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </a-card>
</template>

<script setup>
import { ref, onMounted } from "vue";
import { storeToRefs } from "pinia";
import {
  CheckCircleFilled,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  SortAscendingOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons-vue";
import { theme, message } from "ant-design-vue";
import { Storage } from "@bindings/desktop/storage";
import { AppConfig, OpenExternalURL } from "@bindings/desktop/internal/appconst";
import { Events } from "@wailsio/runtime";
import { useServerStore } from "@/stores/server";
import { useSettingsStore } from "@/stores/settings";
import { t } from "@/locale";
import { useServerPing } from "./servepanel/ServerPing";
import { useServerEditor } from "./servepanel/ServerEditor";
import { useServerSort } from "./servepanel/ServerSort";
import { withRuntimeIds, stripForStorage, stripServerListForStorage } from "@/utils";

const { token } = theme.useToken();

const servers = ref([]);
const serverStore = useServerStore();
const { selectedServer } = storeToRefs(serverStore);

async function loadServers() {
  try {
    const raw = await Storage.GetServers();
    servers.value = withRuntimeIds(stripServerListForStorage(raw));
  } catch {
    message.error(t("serverList.loadFailed"));
  }
}

async function persistServers() {
  await Storage.SetServers(servers.value.map(stripForStorage));
}

function isServerSelected(server) {
  if (!selectedServer.value) return false;
  return selectedServer.value.__id === server?.__id;
}

const settingsStore = useSettingsStore();

async function openSubscribeEditor() {
  const rawHost = (settingsStore.proxy.host || "").trim();
  const host = rawHost === "0.0.0.0" || rawHost === "" ? "127.0.0.1" : rawHost;
  const port = Number(settingsStore.proxy.port);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    message.warning(t("settings.pacNeedPort"));
    return;
  }
  const url = `http://${host}:${port}/subscribe/`;
  try {
    await OpenExternalURL(url);
  } catch (e) {
    message.error(e?.message || t("settings.pacOpenFailed"));
  }
}

const { latencyById, pingingAll, pingAllServers, getLatencyClass } =
  useServerPing(servers);
const { sortByLatency } = useServerSort(servers, latencyById, persistServers);
const { edit, editRef, deleteModal, proxySchemes } = useServerEditor(
  servers,
  latencyById,
  persistServers,
  selectedServer,
);

onMounted(async () => {
  await loadServers();
  Object.assign(latencyById.value, serverStore.restoredLatencies);
  if (serverStore.selectedServer) {
    const match = servers.value.find(
      (s) => s.__id === serverStore.selectedServer.__id,
    );
    if (match) serverStore.selectedServer = match;
  }
  const appConfig = await AppConfig();
  Events.On(appConfig.EventNameServersChanged, loadServers);
});
</script>

<style scoped lang="scss">
.server-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
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
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 12px;
  }

  :deep(.ant-list-item-meta-title) {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  :deep(.ant-list-item-meta-description) {
    min-width: 0;
  }

  .card-extra-actions {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .list {
    flex: 1;
    margin-bottom: 8px;
    margin-right: -12px;
    min-height: 0;
    overflow: hidden auto;
  }

  .empty {
    flex: 1;
    min-height: 120px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .add-server-btn {
    flex-shrink: 0;
  }

  :deep(.ant-list-item) {
    cursor: pointer;
    padding: 12px 20px;

    &.active {
      background: v-bind("token.colorPrimaryBgHover");
    }

    .server-title-row {
      display: flex;
      align-items: center;
      gap: 6px;
      min-width: 0;
    }

    .server-host {
      font-weight: 500;
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .latency-inline {
      flex-shrink: 0;
      margin-left: 6px;
      font-weight: 500;
      font-size: 12px;

      .latency-good {
        color: #52c41a;
      }

      .latency-medium {
        color: #faad14;
      }

      .latency-bad {
        color: #ff4d4f;
      }

      .latency-error {
        color: #ff4d4f;
        font-size: 11px;
      }
    }

    .selected-icon {
      color: #1677ff;
      font-size: 14px;
    }

    .server-meta {
      font-size: 12px;
      color: v-bind("token.colorTextSecondary");
      display: flex;
      align-items: center;
      gap: 4px;
      min-width: 0;
      overflow: hidden;
    }

    .server-protocol,
    .server-separator {
      flex-shrink: 0;
    }

    .server-username {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      display: inline-block;
      max-width: 100%;
    }

  }
}
</style>
