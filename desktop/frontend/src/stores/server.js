import { defineStore } from "pinia";
import { Storage } from "@bindings/desktop/storage";
import { TestServer } from "@bindings/desktop/proxy/proxyping";
import { useSettingsStore } from "@/stores/settings";
import { withRuntimeIds, stripServerListForStorage } from "@/utils";

export const useServerStore = defineStore("server", {
  state: () => ({
    selectedServer: null,
    /** 启动测速结果，供 ServePanel 合并到 latencyById */
    restoredLatencies: {},
  }),

  actions: {
    /** 启动时恢复上次连接的节点（测速后选中，连接由 ActionPanel watch 触发） */
    async init() {
      this.restoredLatencies = {};
      try {
        const stored = await Storage.GetSelectedServer();
        if (!stored?.host || !stored?.protocol) return;

        const raw = await Storage.GetServers();
        const list = withRuntimeIds(stripServerListForStorage(raw));
        const storedId = withRuntimeIds([stored])[0]?.__id;
        const match = list.find((s) => s.__id === storedId);
        if (!match) {
          await Storage.ClearSelectedServer();
          return;
        }

        const settingsStore = useSettingsStore();
        const latencyHost = (settingsStore.proxy.latencyTestHost || "").trim();
        if (latencyHost) {
          const protocol = match.protocol.toLowerCase();
          const username = match.username || "";
          const password = match.password || "";
          const proxyURL = `${protocol}://${username}:${password}@${match.host}`;
          try {
            const result = await TestServer(proxyURL, latencyHost);
            this.restoredLatencies[match.__id] = result.success
              ? result.latency
              : -1;
          } catch {
            this.restoredLatencies[match.__id] = -1;
          }
        }

        this.selectedServer = match;
      } catch {
        // 恢复失败时保持未连接
      }
    },
  },
});
