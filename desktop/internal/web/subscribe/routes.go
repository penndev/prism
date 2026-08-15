package subscribe

import (
	"desktop/internal"
	"desktop/internal/storage"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func HandleServers(w http.ResponseWriter, r *http.Request) {
	st := storage.DefaultStorage
	if st == nil {
		http.Error(w, "storage not initialized", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		servers, err := st.GetServers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(servers)
	case http.MethodPost, http.MethodPut:
		var payload []storage.ServerEntry
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := st.SetServers(payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		emitServersChanged()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleImportServers(w http.ResponseWriter, r *http.Request) {
	st := storage.DefaultStorage
	if st == nil {
		http.Error(w, "storage not initialized", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var imported []storage.ServerEntry
	if err := json.NewDecoder(r.Body).Decode(&imported); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := st.SetServers(imported); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	emitServersChanged()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "count": len(imported)})
}

func HandleExportServers(w http.ResponseWriter, r *http.Request) {
	st := storage.DefaultStorage
	if st == nil {
		http.Error(w, "storage not initialized", http.StatusInternalServerError)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	servers, err := st.GetServers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="subscribe_servers.json"`)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(servers)
}

func HandleSubscriptionConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		URL  string `json:"url"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	subURL := strings.TrimSpace(req.URL)
	if subURL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	subType := strings.ToLower(strings.TrimSpace(req.Type))
	if subType == "" {
		subType = "prism"
	}
	servers, err := convertSubscriptionURL(subURL, subType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"count":   len(servers),
		"servers": servers,
	})
}

func emitServersChanged() {
	if internal.App == nil {
		return
	}
	internal.App.Event.Emit(internal.AppConfig.EventNameServersChanged, time.Now().UnixMilli())
}

