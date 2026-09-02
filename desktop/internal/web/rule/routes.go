package rule

import (
	"desktop/internal"
	"desktop/internal/storage"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/penndev/gopkg/ipregion"
	"github.com/penndev/prism/fakeip"
)

var domainSet = map[string]struct{}{}
var domainMu sync.RWMutex

func setDomainMap(list []string) {
	next := make(map[string]struct{}, len(list))
	for _, d := range list {
		if d != "" {
			next[d] = struct{}{}
		}
	}
	domainMu.Lock()
	domainSet = next
	domainMu.Unlock()
}

func LoadDomains() {
	st := storage.DefaultStorage
	if st == nil {
		return
	}
	cfg, err := st.GetRuleConfig()
	if err != nil || cfg == nil {
		setDomainMap(nil)
		return
	}
	setDomainMap(normalizeDomains(cfg.Domains))

	fakeip.SetNeedFake(func(name string) bool {
		domainMu.RLock()
		defer domainMu.RUnlock()
		if name == "" || len(domainSet) == 0 {
			return false
		}
		for name != "" {
			if _, ok := domainSet[name]; ok {
				return true
			}
			i := strings.IndexByte(name, '.')
			if i < 0 {
				return false
			}
			name = name[i+1:]
		}
		return false
	})

}

type configPayload struct {
	AreaMode string   `json:"areaMode"`
	Mode     string   `json:"mode"` // 兼容旧字段
	AreaIDs  []uint32 `json:"areaIds"`
	Domains  []string `json:"domains"`
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
	setDomainMap(next.Domains)
}

func dbStatusJSON() map[string]any {
	path, err := storage.IpregionDBPath()
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{"path": path, "exists": false, "size": int64(0)}
	if fi, err := os.Stat(path); err == nil {
		out["exists"] = true
		out["size"] = fi.Size()
	}
	if internal.Searcher == nil {
		if s, err := ipregion.Open(path); err == nil {
			internal.Searcher = s
		}
	}
	if internal.Searcher != nil {
		m := internal.Searcher.Meta()
		out["version"] = m.Version
		out["remark"] = m.Remark
		out["areas"] = m.Areas
		out["v4"] = m.V4
		out["v6"] = m.V6
	}
	return out
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
		out := storage.RuleConfig{
			AreaMode: normalizeMode(cfg.AreaMode),
			AreaIDs:  normalizeAreaIDs(cfg.AreaIDs),
			Domains:  normalizeDomains(cfg.Domains),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	case http.MethodPut, http.MethodPost:
		var payload configPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		areaMode := payload.AreaMode
		if strings.TrimSpace(areaMode) == "" {
			areaMode = payload.Mode
		}
		cfg := storage.RuleConfig{
			AreaMode: normalizeMode(areaMode),
			AreaIDs:  normalizeAreaIDs(payload.AreaIDs),
			Domains:  normalizeDomains(payload.Domains),
		}
		if err := st.SetRuleConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setDomainMap(cfg.Domains)
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

type areaNode struct {
	ID       uint32     `json:"id"`
	ParentID uint32     `json:"parent_id"`
	Name     string     `json:"name"`
	Children []areaNode `json:"children,omitempty"`
}

func buildAreaTree(parentID uint32) []areaNode {
	if internal.Searcher == nil {
		return nil
	}
	src := internal.Searcher.Areas(parentID)
	out := make([]areaNode, 0, len(src))
	for _, a := range src {
		out = append(out, areaNode{
			ID:       a.ID,
			ParentID: a.ParentID,
			Name:     a.Name,
			Children: buildAreaTree(a.ID),
		})
	}
	return out
}

func HandleRuleAreas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if internal.Searcher == nil {
		path, err := storage.IpregionDBPath()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s, err := ipregion.Open(path)
		if err != nil {
			http.Error(w, "未找到 ipregion.db，请先下载或上传", http.StatusBadRequest)
			return
		}
		internal.Searcher = s
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(buildAreaTree(0))
}

func HandleRuleDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dbStatusJSON())
}

func HandleRuleDBDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		URL string `json:"url"`
	}
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"db": dbStatusJSON(),
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"db": dbStatusJSON(),
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
