import { message } from "ant-design-vue";
import { t } from "@/locale";

/** 测速结果排序权重：成功 < 失败 < 未测速 */
function latencySortKey(latencyById, id) {
  const latency = latencyById.value[id];
  if (latency === undefined) return [2, Number.POSITIVE_INFINITY];
  if (latency < 0) return [1, Number.POSITIVE_INFINITY];
  return [0, latency];
}

export function useServerSort(servers, latencyById, persistServers) {
  async function sortByLatency() {
    const hasLatencyData = servers.value.some(
      (server) => latencyById.value[server.__id] !== undefined,
    );
    if (!hasLatencyData) {
      message.warning(t("serverList.sortByLatencyNoData"));
      return;
    }

    servers.value = [...servers.value].sort((a, b) => {
      const [rankA, valueA] = latencySortKey(latencyById, a.__id);
      const [rankB, valueB] = latencySortKey(latencyById, b.__id);
      if (rankA !== rankB) return rankA - rankB;
      return valueA - valueB;
    });

    try {
      await persistServers();
      message.success(t("serverList.sortByLatencyDone"));
    } catch {
      message.error(t("serverList.operationFailed"));
    }
  }

  return { sortByLatency };
}
