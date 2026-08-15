package pac

import (
	"desktop/internal/storage"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

type ConfigPayload struct {
	Domains     string `json:"domains"`
	Mode        string `json:"mode"`
	PACTemplate string `json:"pacTemplate"`
}

func HandlePACRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pac/", http.StatusFound)
}

func HandlePACConfig(w http.ResponseWriter, r *http.Request) {
	st := storage.DefaultStorage
	if st == nil {
		http.Error(w, "storage not initialized", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		cfg, err := st.GetPACConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if cfg == nil {
			cfg = &storage.PACConfig{Mode: "proxy"}
		}
		out := *cfg
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(out)
	case http.MethodPut, http.MethodPost:
		var payload ConfigPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		mode := strings.ToLower(strings.TrimSpace(payload.Mode))
		if mode != "proxy" && mode != "direct" {
			http.Error(w, "mode must be proxy or direct", http.StatusBadRequest)
			return
		}
		cfg := storage.PACConfig{
			Domains:     payload.Domains,
			Mode:        mode,
			PACTemplate: payload.PACTemplate,
		}
		if err := st.SetPACConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := buildPACScript(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandlePACScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := storage.DefaultStorage
	if st == nil {
		http.Error(w, "storage not initialized", http.StatusInternalServerError)
		return
	}
	cfg, err := st.GetPACConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		cfg = &storage.PACConfig{Mode: "proxy"}
	}
	script, err := buildPACScript(*cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(script))
}

func buildPACScript(cfg storage.PACConfig) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = "proxy"
	}
	domains := parseDomains(cfg.Domains)
	proxyValue := "PROXY " + getProxyAddress()
	matchAction := "DIRECT"
	defaultAction := proxyValue
	if mode == "proxy" {
		matchAction = proxyValue
		defaultAction = "DIRECT"
	}

	domainsLit, err := json.Marshal(domains)
	if err != nil {
		return "", err
	}
	matchLit, err := json.Marshal(matchAction)
	if err != nil {
		return "", err
	}
	defaultLit, err := json.Marshal(defaultAction)
	if err != nil {
		return "", err
	}

	tmplSrc := strings.TrimSpace(cfg.PACTemplate)
	if tmplSrc == "" {
		return "", errors.New("pacTemplate不能为空")
	}
	tmpl, err := template.New("pac").Parse(tmplSrc)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	data := struct {
		Domains     string
		WhenMatch   string
		WhenDefault string
	}{
		Domains:     string(domainsLit),
		WhenMatch:   string(matchLit),
		WhenDefault: string(defaultLit),
	}
	if err := tmpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func parseDomains(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ' ' || r == '\t'
	})
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, item := range parts {
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

func getProxyAddress() string {
	st := storage.DefaultStorage
	settings, err := st.GetSettings()
	if err != nil || settings == nil {
		return "127.0.0.1:1080"
	}
	host := strings.TrimSpace(settings.Proxy.Host)
	if host == "" || host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	port := settings.Proxy.Port
	if port <= 0 {
		port = 1080
	}
	return host + ":" + strconv.Itoa(port)
}

//go:embed all:static
var staticPACFS embed.FS

func HandlePACFileServer() http.Handler {
	sub, err := fs.Sub(staticPACFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(sub))
}
