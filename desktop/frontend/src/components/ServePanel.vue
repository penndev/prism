<template>
  <a-card class="server-panel" :title="t('serverList.title')">
    <template #extra>
      <div class="card-extra-actions">
        <a-tooltip :title="t('serverList.pingAll')">
          <a-button
            type="text"
            size="small"
            class="card-extra-btn"
            :disabled="pingingAll || servers.length === 0"
            @click="pingAllServers"
          >
            <LoadingOutlined v-if="pingingAll" spin />
            <SignalFilled v-else />
          </a-button>
        </a-tooltip>
        <a-tooltip :title="t('serverList.openSubscribeEditor')">
          <a-button
            type="text"
            size="small"
            class="card-extra-btn"
            @click="openSubscribeEditor"
          >
            <EditFilled />
          </a-button>
        </a-tooltip>
      </div>
    </template>

    <div class="list" v-if="servers.length > 0">
      <a-list :data-source="servers" row-key="_id" bordered>
        <template #renderItem="{ item }">
          <a-list-item
            :class="{ active: isServerSelected(item) }"
            @click="serverStore.selectedServer = stripForStorage(item)"
          >
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
                  <CheckCircleFilled
                    v-if="isServerSelected(item)"
                    class="selected-icon"
                  />
                  <span class="server-host" :title="item.remark || item.host">
                    {{ item.remark || item.host }}
                  </span>
                  <span v-if="item._latency !== undefined" class="latency-inline">
                    <span
                      v-if="item._latency >= 0"
                      :class="getLatencyClass(item._latency)"
                    >
                      {{ item._latency }}ms
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
import { ref, reactive, onMounted, onBeforeUnmount } from "vue";
import {
  CheckCircleFilled,
  DeleteOutlined,
  EditFilled,
  EditOutlined,
  LoadingOutlined,
  PlusOutlined,
  SignalFilled,
} from "@ant-design/icons-vue";
import { theme, Modal, message } from "ant-design-vue";
import { Storage } from "@bindings/desktop/internal/storage";
import {
  AppConfig,
  OpenExternalURL,
  ProxyScheme,
} from "@bindings/desktop/internal/appconst";
import { TestServer } from "@bindings/desktop/internal/proxy/proxyping";
import { Events } from "@wailsio/runtime";
import { useServerStore } from "@/stores/server";
import { useSettingsStore } from "@/stores/settings";
import { t } from "@/locale";
import { extendServerItem, stripForStorage } from "@/utils";

const { token } = theme.useToken();

const serverStore = useServerStore();
const settingsStore = useSettingsStore();

const servers = ref([]);
const pingingAll = ref(false);
const editRef = ref();
const proxySchemes = ref([]);

async function loadServers() {
  try {
    const raw = await Storage.GetServers();
    servers.value = (Array.isArray(raw) ? raw : []).map(extendServerItem);
  } catch {
    message.error(t("serverList.loadFailed"));
  }
}

async function persistServers() {
  await Storage.SetServers(servers.value.map(stripForStorage));
}

function isServerSelected(server) {
  if (!serverStore.selectedServer) return false;
  return extendServerItem(serverStore.selectedServer)._id === server._id;
}

async function pingAllServers() {
  if (servers.value.length === 0) return;

  const latencyHost = (settingsStore.latencyTest.host || "").trim();
  if (!latencyHost) {
    message.warning(t("settings.latencyTestHostRequired"));
    return;
  }

  pingingAll.value = true;
  try {
    const pending = servers.value;
    await Promise.all(
      pending.map(async (server) => {
        let latency = -1;
        try {
          const result = await TestServer(server._id, latencyHost);
          if (result.success) latency = result.latency;
        } catch {
          latency = -1;
        }
        // 按 _id 回写而不是按启动时的下标：测速期间 EventNameServersChanged
        // 可能整体换掉 servers.value，按下标写会让已删除的节点复活。
        const idx = servers.value.findIndex((s) => s._id === server._id);
        if (idx >= 0) {
          servers.value[idx] = { ...servers.value[idx], _latency: latency };
        }
      }),
    );
    if (settingsStore.latencyTest.sortAfterPing) {
      try {
        await applyLatencySort();
      } catch {
        message.error(t("serverList.operationFailed"));
      }
    }
    message.success(t("serverList.pingAllDone"));
  } finally {
    pingingAll.value = false;
  }
}

function getLatencyClass(latency) {
  if (latency < 100) return "latency-good";
  if (latency < 300) return "latency-medium";
  return "latency-bad";
}

// --- 按延迟排序 ---

function latencySortKey(server) {
  const latency = server._latency;
  if (latency === undefined) return [2, Number.POSITIVE_INFINITY];
  if (latency < 0) return [1, Number.POSITIVE_INFINITY];
  return [0, latency];
}

async function applyLatencySort() {
  servers.value = [...servers.value].sort((a, b) => {
    const [rankA, valueA] = latencySortKey(a);
    const [rankB, valueB] = latencySortKey(b);
    if (rankA !== rankB) return rankA - rankB;
    return valueA - valueB;
  });
  await persistServers();
}

// --- 新增 / 编辑 / 删除 ---

const edit = reactive({
  visible: false,
  loading: false,
  title: "",
  editingId: "",
  form: {
    host: "",
    remark: "",
    username: "",
    password: "",
    protocol: "socks5",
  },
  // 用 getter 而不是字面量：字面量里的 t() 只在 setup 时求值一次，
  // 切换语言后校验提示会停留在旧语言。
  get rules() {
    return {
      host: [
        { required: true, message: t("serverList.validateHostRequired") },
        {
          // IPv4/域名: host:port；IPv6: [addr]:port
          pattern: /^(\[[^\]]+\]|[^:\[\]]+):\d{1,5}$/,
          message: t("serverList.validateHostFormat"),
        },
      ],
      protocol: [
        { required: true, message: t("serverList.validateProtocolRequired") },
      ],
    };
  },

  open(server = null) {
    edit.editingId = server?._id ?? "";
    edit.title = edit.editingId
      ? t("serverList.editTitle")
      : t("serverList.addTitle");
    edit.form.host = server?.host ?? "";
    edit.form.remark = server?.remark ?? "";
    edit.form.username = server?.username ?? "";
    edit.form.password = server?.password ?? "";
    edit.form.protocol = server?.protocol ?? "socks5";
    edit.visible = true;
  },

  async submit() {
    try {
      await editRef.value.validate();
      edit.loading = true;

      const payload = {
        host: edit.form.host.trim(),
        remark: edit.form.remark?.trim() ?? "",
        username: edit.form.username?.trim() ?? "",
        password: edit.form.password ?? "",
        protocol: edit.form.protocol,
      };

      // _id 由 protocol/账号/host 派生，两条记录撞上之后编辑和删除都会认错行。
      // 保存前挡掉重复项，列表里就不会出现同 _id 的记录。
      const nextId = extendServerItem(payload)._id;
      const duplicated = servers.value.some(
        (s) => s._id === nextId && s._id !== edit.editingId,
      );
      if (duplicated) {
        message.error(t("serverList.duplicateServer"));
        return;
      }

      const backup = servers.value;
      const selectedWasEdited =
        edit.editingId &&
        serverStore.selectedServer &&
        extendServerItem(serverStore.selectedServer)._id === edit.editingId;

      const next = [...servers.value];
      let editedIdx = -1;
      if (edit.editingId) {
        editedIdx = next.findIndex((s) => s._id === edit.editingId);
        if (editedIdx >= 0) next[editedIdx] = payload;
      } else {
        next.push(payload);
      }
      servers.value = next.map(extendServerItem);

      try {
        await persistServers();
      } catch (err) {
        // 写入失败要还原，否则界面已经改了而存储里还是旧的
        servers.value = backup;
        message.error(err?.message || t("serverList.operationFailed"));
        return;
      }

      if (selectedWasEdited && editedIdx >= 0) {
        serverStore.selectedServer = stripForStorage(servers.value[editedIdx]);
      }
      message.success(
        edit.editingId
          ? t("serverList.updateSuccess")
          : t("serverList.addSuccess"),
      );
      edit.visible = false;
    } catch (e) {
      if (!e?.errorFields) {
        message.error(e?.message || t("serverList.operationFailed"));
      }
    } finally {
      edit.loading = false;
    }
  },
});

function deleteModal(item) {
  Modal.confirm({
    title: t("serverList.deleteTitle"),
    content: `${t("serverList.deleteContentPrefix")}${
      item.remark || item.host
    }${t("serverList.deleteContentSuffix")}`,
    okType: "danger",
    okText: t("serverList.deleteOkText"),
    cancelText: t("serverList.deleteCancelText"),
    async onOk() {
      const backup = servers.value;
      servers.value = servers.value
        .filter((s) => s._id !== item._id)
        .map(extendServerItem);
      try {
        await persistServers();
      } catch (e) {
        // 写入失败要把列表还原，否则界面显示已删除而存储里还在
        servers.value = backup;
        message.error(e?.message || t("serverList.operationFailed"));
        return;
      }
      if (
        serverStore.selectedServer &&
        extendServerItem(serverStore.selectedServer)._id === item._id
      ) {
        serverStore.selectedServer = null;
      }
      message.success(t("serverList.deleteSuccess"));
    },
  });
}

// --- 订阅编辑器 ---

async function openSubscribeEditor() {
  if (!settingsStore.webBaseURL) {
    message.warning(t("settings.pacNeedPort"));
    return;
  }
  try {
    await OpenExternalURL(`${settingsStore.webBaseURL}/subscribe/`);
  } catch (e) {
    message.error(e?.message || t("settings.pacOpenFailed"));
  }
}

let offServersChanged = null;

onMounted(async () => {
  await loadServers();
  proxySchemes.value = await ProxyScheme();

  const appConfig = await AppConfig();
  offServersChanged = Events.On(appConfig.EventNameServersChanged, loadServers);
});

onBeforeUnmount(() => {
  offServersChanged?.();
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
