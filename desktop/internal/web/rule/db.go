package rule

import (
	"desktop/internal"
	"desktop/internal/lang"
	"desktop/internal/storage"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/penndev/gopkg/ipregion"
)

const maxDBBytes = 64 << 20

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
		http.Error(w, lang.DefaultLang.T("rules.db.missingFile"), http.StatusBadRequest)
		return
	}
	defer file.Close()
	if err := saveUploadedDBFile(file); err != nil {
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
	_ = internal.EnsureSearcher(path)
	searcher := internal.AcquireSearcher()
	defer internal.ReleaseSearcher()
	if searcher != nil {
		m := searcher.Meta()
		out["version"] = m.Version
		out["remark"] = m.Remark
		out["areas"] = m.Areas
		out["v4"] = m.V4
		out["v6"] = m.V6
	}
	return out
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
	path, err := storage.IpregionDBPath()
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
	return swapDBFile(tmp, path)
}

func saveUploadedDBFile(r io.Reader) error {
	path, err := storage.IpregionDBPath()
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
	return swapDBFile(tmp, path)
}

// swapDBFile 用校验通过的 tmp 覆盖当前库并重新打开。
// Windows 不允许 rename 覆盖仍被打开的文件，所以必须先关掉当前 Searcher，
// 否则只要规则页加载过一次 DB 状态，后续所有下载/上传都会以「拒绝访问」失败。
func swapDBFile(tmp, path string) error {
	if err := validateDBFile(tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	internal.SetSearcher(nil)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s, err := ipregion.Open(path, ipregion.ModeMemory)
	if err != nil {
		return err
	}
	internal.SetSearcher(s)
	return nil
}

func validateDBFile(path string) error {
	s, err := ipregion.Open(path, ipregion.ModeMemory)
	if err != nil {
		return fmt.Errorf("%s: %w", lang.DefaultLang.T("rules.db.invalid"), err)
	}
	_ = s.Close()
	return nil
}
