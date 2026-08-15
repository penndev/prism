// Package ipregion 封装应用内 ipregion.db 的打开与查询，供规则页与连接分流共用。
// 库文件的下载/上传由 web 层负责写入 DBPath 后调用 Reset。
package ipregion

import (
	"desktop/storage"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	gopkg "github.com/penndev/gopkg/ipregion"
)

const DBName = "ipregion.db"

// Status 库文件状态。
type Status struct {
	Exists  bool   `json:"exists"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Version string `json:"version,omitempty"`
	Remark  string `json:"remark,omitempty"`
	Areas   int    `json:"areas,omitempty"`
	V4      int    `json:"v4,omitempty"`
	V6      int    `json:"v6,omitempty"`
}

var (
	mu       sync.Mutex
	searcher *gopkg.Searcher
)

func DBPath() (string, error) {
	dir, err := storage.AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DBName), nil
}

// Reset 关闭并清空缓存的 Searcher（库文件更新后调用）。
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	if searcher != nil {
		_ = searcher.Close()
		searcher = nil
	}
}

// Get 返回已打开的 Searcher，未打开时为 nil。
func Get() *gopkg.Searcher {
	mu.Lock()
	defer mu.Unlock()
	return searcher
}

// Open 打开应用配置目录下的 ipregion.db（带缓存）。
func Open() (*gopkg.Searcher, error) {
	mu.Lock()
	defer mu.Unlock()
	if searcher != nil {
		return searcher, nil
	}
	path, err := DBPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("未找到 ipregion.db，请先下载或上传")
	}
	s, err := gopkg.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开 ipregion.db 失败: %w", err)
	}
	searcher = s
	return searcher, nil
}

// StatusOf 返回库状态。
func StatusOf() (Status, error) {
	path, err := DBPath()
	if err != nil {
		return Status{}, err
	}
	out := Status{Path: path}
	fi, err := os.Stat(path)
	if err != nil {
		return out, nil
	}
	out.Exists = true
	out.Size = fi.Size()
	if s, err := Open(); err == nil && s != nil {
		m := s.Meta()
		out.Version = m.Version
		out.Remark = m.Remark
		out.Areas = m.Areas
		out.V4 = m.V4
		out.V6 = m.V6
	}
	return out, nil
}

// Find 查询 IP 地域信息。
func Find(ip string) (gopkg.Info, error) {
	s, err := Open()
	if err != nil {
		return gopkg.Info{}, err
	}
	return s.Find(ip)
}

// Names 按地域 ID 查询名称；找不到的 ID 会被跳过。
func Names(ids []uint32) []string {
	out := make([]string, 0, len(ids))
	if len(ids) == 0 {
		return out
	}
	s, err := Open()
	if err != nil {
		return out
	}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		a, ok := s.Area(id)
		if !ok || a.Name == "" {
			continue
		}
		out = append(out, a.Name)
	}
	return out
}

// InAreas 判断 address（host 或 host:port）是否属于给定地域 ID（含上级）。
// 无法解析或不在列表中时返回 false。
func InAreas(address string, areaIDs []uint32) bool {
	if len(areaIDs) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return false
		}
		ip = ips[0]
	}
	info, err := Find(ip.String())
	if err != nil || info.Area.ID == 0 {
		return false
	}
	set := make(map[uint32]struct{}, len(areaIDs))
	for _, id := range areaIDs {
		if id != 0 {
			set[id] = struct{}{}
		}
	}
	if len(set) == 0 {
		return false
	}
	for a := &info.Area; a != nil; a = a.Parent {
		if _, ok := set[a.ID]; ok {
			return true
		}
	}
	return false
}
