package rule

import (
	"desktop/internal/storage"
	"encoding/json"
	"net/http"
	"strings"
)

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
		var payload storage.RuleConfig
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		cfg := storage.RuleConfig{
			AreaMode: normalizeMode(payload.AreaMode),
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

func normalizeMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "proxy", "bypass", "none":
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
