package rule

import (
	"desktop/internal"
	"desktop/ipregion"
	"desktop/storage"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

type configPayload struct {
	Mode    string   `json:"mode"`
	AreaIDs []uint32 `json:"areaIds"`
}

type downloadPayload struct {
	URL string `json:"url"`
}

func HandleRuleRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/rule/", http.StatusFound)
}

func normalizeMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "proxy", "bypass":
		return mode
	default:
		return "global"
	}
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

func normalizeConfig(mode string, ids []uint32) storage.RuleConfig {
	return storage.RuleConfig{
		Mode:    normalizeMode(mode),
		AreaIDs: normalizeAreaIDs(ids),
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
	_ = st.SetRuleConfig(storage.RuleConfig{
		Mode:    "global",
		AreaIDs: []uint32{},
	})
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
			cfg = &storage.RuleConfig{Mode: "global", AreaIDs: []uint32{}}
		}
		out := normalizeConfig(cfg.Mode, cfg.AreaIDs)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"mode":      out.Mode,
			"areaIds":   out.AreaIDs,
			"areaNames": ipregion.Names(out.AreaIDs),
		})
	case http.MethodPut, http.MethodPost:
		var payload configPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		cfg := normalizeConfig(payload.Mode, payload.AreaIDs)
		if err := st.SetRuleConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		emitRuleChanged()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "mode": cfg.Mode, "areaIds": cfg.AreaIDs})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleRuleAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s, err := ipregion.Open()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(s.Areas(0))
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
