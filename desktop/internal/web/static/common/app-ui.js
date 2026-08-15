window.WebAppUI = (function () {
  const APP_CONFIG_DEFAULT = {
    language: 'zh-CN',
    themeMode: 'system'
  };

  function formatI18n(text, params) {
    if (!params || typeof params !== 'object') return text;
    return String(text).replace(/\{(\w+)\}/g, function (_, token) {
      return Object.prototype.hasOwnProperty.call(params, token) ? String(params[token]) : '';
    });
  }

  function createTranslator(i18nTable, defaultLang) {
    const fallback = defaultLang || 'zh-CN';
    return function (lang, key, params) {
      const table = i18nTable[lang] || i18nTable[fallback] || {};
      return formatI18n(table[key] || key, params);
    };
  }

  function normalizeLanguage(i18nTable, value, fallback) {
    const lang = String(value || '').trim();
    return i18nTable[lang] ? lang : (fallback || 'zh-CN');
  }

  function getSystemTheme() {
    if (!window.matchMedia) return 'light';
    const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    return isDark ? 'dark' : 'light';
  }

  var systemThemeListenerOff = null;

  function applyTheme(mode) {
    if (systemThemeListenerOff) {
      systemThemeListenerOff();
      systemThemeListenerOff = null;
    }

    const normalized = String(mode || 'system').toLowerCase();
    const resolved =
      normalized === 'dark'
        ? 'dark'
        : normalized === 'light'
          ? 'light'
          : getSystemTheme();

    document.documentElement.setAttribute('data-theme', resolved);

    if (normalized === 'system' && window.matchMedia) {
      const mql = window.matchMedia('(prefers-color-scheme: dark)');
      var handler = function () {
        document.documentElement.setAttribute('data-theme', getSystemTheme());
      };
      mql.addEventListener('change', handler);
      systemThemeListenerOff = function () {
        mql.removeEventListener('change', handler);
      };
    }

    return resolved;
  }

  async function fetchAppConfig(requestFn) {
    try {
      const cfg = await requestFn('/api/app-config');
      return {
        language: String(cfg.language || APP_CONFIG_DEFAULT.language),
        themeMode: String(cfg.themeMode || APP_CONFIG_DEFAULT.themeMode)
      };
    } catch (_) {
      return APP_CONFIG_DEFAULT;
    }
  }

  return {
    APP_CONFIG_DEFAULT: APP_CONFIG_DEFAULT,
    formatI18n: formatI18n,
    createTranslator: createTranslator,
    normalizeLanguage: normalizeLanguage,
    applyTheme: applyTheme,
    fetchAppConfig: fetchAppConfig
  };
})();
