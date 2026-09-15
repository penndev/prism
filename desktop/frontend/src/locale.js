import { ref } from "vue";
import { Bundle, SetLocale } from "@bindings/desktop/internal/lang/lang";
import { Events } from "@wailsio/runtime";
import { AppConfig } from "@bindings/desktop/internal/appconst";

export const LANGUAGE_SYSTEM = "system";

// 语言文件对象
export const languageLocale = ref({});

export async function applyLocale(language) {
  languageLocale.value = await Bundle(language);
  await SetLocale(language);
}

// 监听语言改变事件
export async function subscribeLocaleEvents(language) {
  await applyLocale(language);
  const appConst = await AppConfig();
  Events.On(appConst.EventNameLocaleChanged, async (ev) => {
    languageLocale.value = await Bundle(ev.data);
  });
}

/**
 * 
 * @param {string} key 
 * @returns {string}
 */
export function t(key) {
  return languageLocale.value[key] ?? key;
}
