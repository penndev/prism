import { defineStore } from "pinia";
import { Storage } from "@bindings/desktop/internal/storage";
import { notification } from "ant-design-vue";
import { debounce } from "@/utils";
import { t, subscribeLocaleEvents, languageLocale } from "@/locale";
import { Bundle } from "@bindings/desktop/internal/lang/lang";
import { Enable, Disable } from "@bindings/desktop/internal/autostart/autostart";

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
  }),

  actions: {
    /** 将 state 持久化到存储 */
    async save() {
      try {
        await Storage.SetSettings({
          proxy: this.proxy,
          latencyTest: this.latencyTest,
          system: this.system,
        });
        // 设置切换语言环境
        languageLocale.value = await Bundle(
          this.system.language
        );
        if (this.system.startupOnBoot) {
          await Enable();
        } else {
          await Disable();
        }
        notification.success({
          message: t("settings.saveSuccess"),
          placement: "topRight",
        });
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
            host:
              storedLatencyTest.host ??
              storedProxy.latencyTestHost ??
              this.latencyTest.host,
            sortAfterPing:
              storedLatencyTest.sortAfterPing ??
              storedProxy.sortByLatencyAfterPing ??
              this.latencyTest.sortAfterPing,
          };

          this.system = {
            ...this.system,
            ...(storedSettings.system ?? {}),
          };
        }else{
          // 强制设置默认语言环境
          if(navigator.language.startsWith("zh")){
            this.system.language = "zh-CN";
          }else{
            this.system.language = "en";
          }
        }
        const save = debounce(this.save, 800)
        this.$subscribe(save);
        // 设置初始语言与开机启动
        console.log("初始语言", this.system.language);
        await subscribeLocaleEvents(this.system.language);
        if (this.system.startupOnBoot) {
          await Enable();
        } else {
          await Disable();
        }
      } catch (error) {
        notification.error({
          message: t("settings.loadError"),
          placement: "topRight",
        });
      }
    },
  },
});
