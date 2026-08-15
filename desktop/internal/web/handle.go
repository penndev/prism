package web

import (
	"desktop/internal/storage"
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

func handleAppConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := storage.DefaultStorage
	if st == nil {
		http.Error(w, "storage not initialized", http.StatusInternalServerError)
		return
	}
	settings, err := st.GetSettings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	language := "zh-CN"
	themeMode := "system"
	if settings != nil {
		if v := strings.TrimSpace(settings.System.Language); v != "" {
			language = v
		}
		if v := strings.TrimSpace(settings.System.ThemeMode); v != "" {
			themeMode = strings.ToLower(v)
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"language":  language,
		"themeMode": themeMode,
	})
}

func handleRoot(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("/"))
}

//go:embed all:static/common
var staticCommonFS embed.FS

func handleCommonFileServer() http.Handler {
	sub, err := fs.Sub(staticCommonFS, "static/common")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
