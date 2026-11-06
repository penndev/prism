package rule

import (
	"desktop/internal/storage"
	"encoding/json"
	"net/http"
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
			cfg = &storage.RuleConfig{AreaMode: "global", AreaIDs: []uint32{}, Domains: ""}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(cfg)
	case http.MethodPut, http.MethodPost:
		var payload storage.RuleConfig
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := st.SetRuleConfig(payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		setDomainMap(payload.Domains)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":       true,
			"areaMode": payload.AreaMode,
			"areaIds":  payload.AreaIDs,
			"domains":  payload.Domains,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func resetRuleToGlobal() {
	st := storage.DefaultStorage
	if st == nil {
		return
	}
	next := storage.RuleConfig{
		AreaMode: "global",
		AreaIDs:  []uint32{},
		Domains:  "",
	}
	if cfg, err := st.GetRuleConfig(); err == nil && cfg != nil {
		next.Domains = cfg.Domains
	}
	_ = st.SetRuleConfig(next)
	setDomainMap(next.Domains)
}
