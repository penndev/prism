package rule

import (
	"desktop/internal"
	"desktop/internal/storage"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/penndev/gopkg/ipregion"
)

func downloadDBFile(rawURL string) error {
	url := strings.TrimSpace(rawURL)
	if url == "" {
		return fmt.Errorf("请填写下载地址")
	}
	path, err := storage.IpregionDBPath()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	_ = os.Remove(tmp)

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := validateDBFile(tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s, err := ipregion.Open(path)
	if err != nil {
		return err
	}
	internal.Searcher = s
	return nil
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
	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := validateDBFile(tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	s, err := ipregion.Open(path)
	if err != nil {
		return err
	}
	internal.Searcher = s
	return nil
}

func validateDBFile(path string) error {
	s, err := ipregion.Open(path)
	if err != nil {
		return fmt.Errorf("无效的 ipregion.db: %w", err)
	}
	_ = s.Close()
	return nil
}
