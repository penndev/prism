package rule

import (
	"desktop/internal"
	"desktop/internal/ipregion"
	"desktop/internal/storage"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

type configPayload struct {
	AreaMode string   `json:"areaMode"`
	Mode     string   `json:"mode"` // 兼容旧字段
	AreaIDs  []uint32 `json:"areaIds"`
	Domains  []string `json:"domains"`
}

func (p configPayload) areaMode() string {
	if strings.TrimSpace(p.AreaMode) != "" {
		return p.AreaMode
	}
	return p.Mode
}

type downloadPayload struct {
	URL string `json:"url"`
}

func HandleRuleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/rule/", http.StatusFound)
}

func normalizeMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "proxy", "bypass", "none":
		return mode
	default:
		return "global"
	}
}

func normalizeDomains(list []string) []string {
	if list == nil {
		return []string{}
	}
	out := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		d := strings.ToLower(strings.TrimSpace(item))
		d = strings.TrimPrefix(d, ".")
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	return out
}

func normalizeAreaIDs(ids []uint32) []uint32 {
	if ids == nil {
		return []uint32{}
	}
	out := make([]uint32, 0, len(ids))
	seen := make(map[uint32]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeConfig(areaMode string, ids []uint32, domains []string) storage.RuleConfig {
	return storage.RuleConfig{
		AreaMode: normalizeMode(areaMode),
		AreaIDs:  normalizeAreaIDs(ids),
		Domains:  normalizeDomains(domains),
	}
}

func encodeConfig(cfg storage.RuleConfig) map[string]any {
	return map[string]any{
		"areaMode":  cfg.AreaMode,
		"areaIds":   cfg.AreaIDs,
		"areaNames": ipregion.Names(cfg.AreaIDs),
		"domains":   cfg.Domains,
	}
}

func emitRuleChanged() {
	if internal.App == nil {
		return
	}
	internal.App.Event.Emit(internal.AppConfig.EventNameRuleChanged, time.Now().UnixMilli())
}

// resetRuleToGlobal 库文件更新后切回全局代理并清空已选地域。
func resetRuleToGlobal() {
	st := storage.DefaultStorage
	if st == nil {
		return
	}
	next := storage.RuleConfig{
		AreaMode: "global",
		AreaIDs:  []uint32{},
		Domains:  []string{},
	}
	if cfg, err := st.GetRuleConfig(); err == nil && cfg != nil {
		next.Domains = normalizeDomains(cfg.Domains)
	}
	_ = st.SetRuleConfig(next)
}

func dbStatusJSON(st ipregion.Status) map[string]any {
	return map[string]any{
		"exists":  st.Exists,
		"path":    st.Path,
		"size":    st.Size,
		"version": st.Version,
		"remark":  st.Remark,
		"areas":   st.Areas,
		"v4":      st.V4,
		"v6":      st.V6,
	}
}

func HandleRuleConfig(w http.ResponseWriter, r *http.Request) {
	st := storage.DefaultStorage
	if st == nil {
		http.Error(w, "storage not initialized", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := st.GetRuleConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if cfg == nil {
			cfg = &storage.RuleConfig{AreaMode: "global", AreaIDs: []uint32{}, Domains: []string{}}
		}
		out := normalizeConfig(cfg.AreaMode, cfg.AreaIDs, cfg.Domains)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(encodeConfig(out))
	case http.MethodPut, http.MethodPost:
		var payload configPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		cfg := normalizeConfig(payload.areaMode(), payload.AreaIDs, payload.Domains)
		if err := st.SetRuleConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		emitRuleChanged()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":       true,
			"areaMode": cfg.AreaMode,
			"areaIds":  cfg.AreaIDs,
			"domains":  cfg.Domains,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleRuleAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tree, err := ipregion.AreaTree()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tree)
}

func HandleRuleDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st, err := ipregion.StatusOf()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dbStatusJSON(st))
}

func HandleRuleDBDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload downloadPayload
	_ = json.NewDecoder(r.Body).Decode(&payload)
	url := strings.TrimSpace(payload.URL)
	if url == "" {
		http.Error(w, "请填写下载地址", http.StatusBadRequest)
		return
	}
	if err := downloadDBFile(url); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resetRuleToGlobal()
	emitRuleChanged()
	status, _ := ipregion.StatusOf()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"db": dbStatusJSON(status),
	})
}

func HandleRuleDBUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "缺少 file 字段", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if err := saveUploadedDBFile(io.Reader(file)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resetRuleToGlobal()
	emitRuleChanged()
	status, _ := ipregion.StatusOf()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"db": dbStatusJSON(status),
	})
}

//go:embed all:static
var staticRuleFS embed.FS

func HandleRuleFileServer() http.Handler {
	sub, err := fs.Sub(staticRuleFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
