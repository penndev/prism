package rule

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"desktop/internal/ipregion"
	"desktop/internal/lang"

	"github.com/penndev/prism/engine"
)

const maxDBBytes = 64 << 20

func HandleRuleDB(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ipregion.Read())
	case http.MethodDelete:
		if err := ipregion.Remove(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resetGeoToDefault()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"db": ipregion.Read(),
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func HandleRuleDBDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	addr := strings.TrimSpace(payload.URL)
	if addr == "" {
		http.Error(w, lang.DefaultLang.T("rules.db.urlRequired"), http.StatusBadRequest)
		return
	}
	if err := downloadDBFile(addr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resetGeoToDefault()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"db": ipregion.Read(),
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
		http.Error(w, lang.DefaultLang.T("rules.db.missingFile"), http.StatusBadRequest)
		return
	}
	defer file.Close()
	if err := saveUploadedDBFile(file); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resetGeoToDefault()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"db": ipregion.Read(),
	})
}

func downloadDBFile(rawURL string) error {
	addr := strings.TrimSpace(rawURL)
	if addr == "" {
		return errors.New(lang.DefaultLang.T("rules.db.urlRequired"))
	}
	parsed, err := url.Parse(addr)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New(lang.DefaultLang.T("rules.db.badURL"))
	}
	path, err := ipregion.Path()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(addr)
	if err != nil {
		return fmt.Errorf("%s: %w", lang.DefaultLang.T("rules.db.downloadFailed"), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", lang.DefaultLang.T("rules.db.downloadFailed"), resp.StatusCode)
	}

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, maxDBBytes+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n > maxDBBytes {
		_ = os.Remove(tmp)
		return errors.New(lang.DefaultLang.T("rules.db.tooLarge"))
	}
	return swapDBFile(tmp)
}

func saveUploadedDBFile(r io.Reader) error {
	path, err := ipregion.Path()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, io.LimitReader(r, maxDBBytes+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n > maxDBBytes {
		_ = os.Remove(tmp)
		return errors.New(lang.DefaultLang.T("rules.db.tooLarge"))
	}
	return swapDBFile(tmp)
}

func swapDBFile(tmp string) error {
	if err := engine.CheckIpregionDB(tmp); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s: %w", lang.DefaultLang.T("rules.db.invalid"), err)
	}
	return ipregion.Replace(tmp)
}
