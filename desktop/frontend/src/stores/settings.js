import { defineStore } from "pinia";
import { Storage } from "@bindings/desktop/internal/storage";
import { notification } from "ant-design-vue";
import { debounce } from "@/utils";
import { t, subscribeLocaleEvents } from "@/locale";
import { SetLocale } from "@bindings/desktop/internal/lang/lang";
import { Enable, Disable } from "@bindings/desktop/internal/autostart/autostart";

// 已生效的值。save() 每次改动都会跑，只有这两项真正变化时才需要动后端语言和注册表启动项。
let appliedLanguage = null;
let appliedStartupOnBoot = null;

export const useSettingsStore = defineStore("settings", {
  state: () => ({
    proxy: {
      host: "127.0.0.1",
      port: 1080,
      username: "",
      password: "",
    },
    latencyTest: {
      host: "google.com",
      sortAfterPing: true,
    },
    system: {
      language: '',
      themeMode: "system",
      startupOnBoot: false,
      enableLogRecording: false,
    },
    proxyMode: "manual",
  }),

  getters: {
    /** 本地代理 Web 服务根地址（规则页/订阅页都用它）；端口非法时为空串 */
    webBaseURL(state) {
      const rawHost = (state.proxy.host || "").trim();
      const host =
        rawHost === "0.0.0.0" || rawHost === "" ? "127.0.0.1" : rawHost;
      const port = Number(state.proxy.port);
      if (!Number.isInteger(port) || port < 1 || port > 65535) {
        return "";
      }
      return `http://${host}:${port}`;
    },
  },

  actions: {
    /** 将 state 持久化到存储 */
    async save() {
      try {
        await Storage.SetSettings({
          proxy: this.proxy,
          latencyTest: this.latencyTest,
          system: this.system,
          proxyMode: this.proxyMode,
        });
        if (this.system.language !== appliedLanguage) {
          appliedLanguage = this.system.language;
          // 前端文案和托盘文案都由后端的 localeChanged 事件驱动，这里只负责通知后端
          await SetLocale(this.system.language);
        }
        if (this.system.startupOnBoot !== appliedStartupOnBoot) {
          appliedStartupOnBoot = this.system.startupOnBoot;
          await (this.system.startupOnBoot ? Enable() : Disable());
        }
      } catch (error) {
        notification.error({
          message: t("settings.saveError"),
          placement: "topRight",
        });
      }
    },

    /** 从存储加载并合并到 state */
    async init() {
      try {
        const storedSettings = await Storage.GetSettings();
        if (storedSettings) {
          const storedProxy = storedSettings.proxy ?? {};
          this.proxy = {
            ...this.proxy,
            host: storedProxy.host ?? this.proxy.host,
            port: storedProxy.port ?? this.proxy.port,
            username: storedProxy.username ?? this.proxy.username,
            password: storedProxy.password ?? this.proxy.password,
          };

          const storedLatencyTest = storedSettings.latencyTest ?? {};
          this.latencyTest = {
            ...this.latencyTest,
            host: storedLatencyTest.host ?? this.latencyTest.host,
            sortAfterPing:
              storedLatencyTest.sortAfterPing ?? this.latencyTest.sortAfterPing,
          };

          this.system = {
            ...this.system,
            ...(storedSettings.system ?? {}),
          };

          this.proxyMode = storedSettings.proxyMode || this.proxyMode;
        }else{
          // 强制设置默认语言环境
          if(navigator.language.startsWith("zh")){
            this.system.language = "zh-CN";
          }else{
            this.system.language = "en";
          }
        }
        // 设置初始语言与开机启动
        appliedLanguage = this.system.language;
        appliedStartupOnBoot = this.system.startupOnBoot;
        await subscribeLocaleEvents(this.system.language);
        await (this.system.startupOnBoot ? Enable() : Disable());
        this.$subscribe(debounce(this.save, 800));
      } catch (error) {
        notification.error({
          message: t("settings.loadError"),
          placement: "topRight",
        });
      }
    },
  },
});
