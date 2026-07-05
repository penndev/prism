import { defineStore } from "pinia";
import { Storage } from "@bindings/desktop/storage";
import { stripForStorage } from "@/utils";

export const useServerStore = defineStore("server", {
  state: () => ({
    /** 当前选中节点的持久化字段（不含 _id、_latency） */
    selectedServer: null,
  }),

  actions: {
    /** 启动时恢复上次选中的节点；连接由 ActionPanel watch 触发 */
    async init() {
      try {
        const stored = await Storage.GetSelectedServer();
        if (!stored?.host || !stored?.protocol) return;
        this.selectedServer = stripForStorage(stored);
      } catch {
        // 恢复失败时保持未连接
      }
    },
  },
});
