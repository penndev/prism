package engine

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	"github.com/penndev/gopkg/ipregion"
)

// Area is a node in the IP-region tree. ParentID 0 is a top-level region.
type Area struct {
	ID       int64
	ParentID int64
	Name     string
}

// AreaList is a gomobile-friendly wrapper; slices of pointers are skipped by gobind.
type AreaList struct {
	items []*Area
}

func (l *AreaList) Len() int64 {
	if l == nil {
		return 0
	}
	return int64(len(l.items))
}

func (l *AreaList) Get(i int64) *Area {
	if l == nil || i < 0 || int(i) >= len(l.items) {
		return nil
	}
	return l.items[int(i)]
}

// DbStatus is the current ipregion.db file and metadata.
type DbStatus struct {
	Exists  bool
	Path    string
	Size    int64
	Version string
	Remark  string
	Areas   int64
	V4      int64
	V6      int64
}

type areaNode struct {
	ID       uint32     `json:"id"`
	ParentID uint32     `json:"parent_id"`
	Name     string     `json:"name"`
	Children []areaNode `json:"children,omitempty"`
}

// 库句柄由 Java 线程（换库）和每条连接的 Lookup 同时访问，必须加锁：
// 换库时把正在 Find 的句柄关掉就是 use-after-close。
var (
	geoMu    sync.RWMutex
	dbPath   string
	searcher *ipregion.Searcher
)

// SetIpregionDB opens ipregion.db at path. Java downloads/copies the file first.
func SetIpregionDB(path string) error {
	s, err := ipregion.Open(path)
	if err != nil {
		return err
	}
	geoMu.Lock()
	old := searcher
	searcher = s
	dbPath = path
	geoMu.Unlock()
	if old != nil && old != s {
		_ = old.Close()
	}
	return nil
}

func DBStatus() *DbStatus {
	geoMu.RLock()
	defer geoMu.RUnlock()

	out := &DbStatus{Path: dbPath}
	if dbPath != "" {
		if fi, err := os.Stat(dbPath); err == nil {
			out.Exists = true
			out.Size = fi.Size()
		}
	}
	if searcher == nil {
		return out
	}
	m := searcher.Meta()
	out.Version = m.Version
	out.Remark = m.Remark
	out.Areas = int64(m.Areas)
	out.V4 = int64(m.V4)
	out.V6 = int64(m.V6)
	return out
}

// AreaTree returns the full region tree as JSON, same shape as desktop /rule/api/areas.
func AreaTree() string {
	geoMu.RLock()
	defer geoMu.RUnlock()
	if searcher == nil {
		return "[]"
	}
	raw, err := json.Marshal(buildAreaTree(searcher, 0))
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// buildAreaTree 递归，所以 searcher 由调用方取好传进来，避免在递归里反复加锁。
func buildAreaTree(s *ipregion.Searcher, parentID uint32) []areaNode {
	src := s.Areas(parentID)
	out := make([]areaNode, 0, len(src))
	for _, a := range src {
		out = append(out, areaNode{
			ID:       a.ID,
			ParentID: a.ParentID,
			Name:     a.Name,
			Children: buildAreaTree(s, a.ID),
		})
	}
	return out
}

// Lookup returns the area chain for address (host or host:port), leaf first then parents.
func Lookup(address string) *AreaList {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		cancel()
		if err != nil || len(ips) == 0 {
			return &AreaList{}
		}
		ip = ips[0]
	}

	geoMu.RLock()
	defer geoMu.RUnlock()
	if searcher == nil {
		return &AreaList{}
	}
	info, err := searcher.Find(ip.String())
	if err != nil || info.Area.ID == 0 {
		return &AreaList{}
	}
	out := make([]*Area, 0, 8)
	for a := &info.Area; a != nil; a = a.Parent {
		pid := int64(0)
		if a.Parent != nil {
			pid = int64(a.Parent.ID)
		}
		out = append(out, &Area{
			ID:       int64(a.ID),
			ParentID: pid,
			Name:     a.Name,
		})
	}
	return &AreaList{items: out}
}
