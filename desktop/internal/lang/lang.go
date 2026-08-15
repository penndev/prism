// Package lang 从嵌入的 JSON 提供多语言文案，并通过 Wails 绑定暴露给前端。
package lang

import (
	"desktop/internal"
	"embed"
	"encoding/json"
	"errors"
	"path"
	"sort"
	"sync"
)

//go:embed locales/*.json
var localeFS embed.FS

// Lang 绑定服务：文案来自 locales 目录下的 JSON。
type Lang struct {
	mu      sync.RWMutex
	locale  string
	bundles map[string]map[string]string
}

var DefaultLang *Lang

// New 加载全部嵌入的语言 JSON。
func New() (*Lang, error) {
	bundles := make(map[string]map[string]string)
	entries, err := localeFS.ReadDir("locales")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := localeFS.ReadFile(path.Join("locales", e.Name()))
		if err != nil {
			return nil, err
		}
		id := e.Name()
		if len(id) > 5 && id[len(id)-5:] == ".json" {
			id = id[:len(id)-5]
		}
		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		bundles[id] = m
	}
	l := &Lang{
		locale:  "en",
		bundles: bundles,
	}
	if _, ok := bundles[l.locale]; !ok {
		for id := range bundles {
			l.locale = id
			break
		}
	}
	return l, nil
}

// AvailableLocales 返回已嵌入的语言标识列表（排序后）。
func (l *Lang) AvailableLocales() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]string, 0, len(l.bundles))
	for id := range l.bundles {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// CurrentLocale 返回当前语言。
func (l *Lang) CurrentLocale() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.locale
}

// SetLocale 切换语言并向前端派发 localeChanged 事件。
func (l *Lang) SetLocale(locale string) error {
	if l.locale == locale {
		return nil
	}
	l.mu.Lock()
	if _, ok := l.bundles[locale]; !ok {
		l.mu.Unlock()
		return errors.New("locale not found: " + locale)
	}
	l.locale = locale
	l.mu.Unlock()
	if internal.App != nil {
		internal.App.Event.Emit(internal.AppConfig.EventNameLocaleChanged, locale)
	}
	return nil
}

// T 按当前语言翻译 key；缺失时返回 key。
func (l *Lang) T(key string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	m := l.bundles[l.locale]
	if m == nil {
		return key
	}
	if v, ok := m[key]; ok {
		return v
	}
	return key
}

// Bundle 返回指定语言的完整键值表（用于前端一次性注入）；未知语言返回空 map。
func (l *Lang) Bundle(locale string) map[string]string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	src := l.bundles[locale]
	if src == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
