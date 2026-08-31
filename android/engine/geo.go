package engine

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	gopkg "github.com/penndev/gopkg/ipregion"
)

var (
	geoMu    sync.Mutex
	searcher *gopkg.Searcher
	dbPath   string
)

type areaNode struct {
	ID       uint32     `json:"id"`
	ParentID uint32     `json:"parent_id"`
	Name     string     `json:"name"`
	Children []areaNode `json:"children,omitempty"`
}

// SetIpregionDB opens ipregion.db at path. Java downloads/copies the file first.
// On error the previously opened DB is left unchanged.
func SetIpregionDB(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("db path required")
	}
	s, err := gopkg.Open(path)
	if err != nil {
		return fmt.Errorf("open ipregion.db: %w", err)
	}
	geoMu.Lock()
	defer geoMu.Unlock()
	if searcher != nil {
		_ = searcher.Close()
	}
	searcher = s
	dbPath = path
	return nil
}

// DBStatus returns the last opened file's metadata. Path is empty if never opened.
func DBStatus() *DbStatus {
	geoMu.Lock()
	path := dbPath
	s := searcher
	geoMu.Unlock()

	out := &DbStatus{Path: path}
	if path == "" {
		return out
	}
	fi, err := os.Stat(path)
	if err != nil {
		return out
	}
	out.Exists = true
	out.Size = fi.Size()
	if s != nil {
		m := s.Meta()
		out.Version = m.Version
		out.Remark = m.Remark
		out.Areas = int64(m.Areas)
		out.V4 = int64(m.V4)
		out.V6 = int64(m.V6)
	}
	return out
}

// AreaTree returns the full region tree as JSON, same shape as desktop /rule/api/areas.
func AreaTree() string {
	geoMu.Lock()
	s := searcher
	geoMu.Unlock()
	if s == nil {
		return "[]"
	}
	raw, err := json.Marshal(buildAreaTree(s, 0))
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func buildAreaTree(s *gopkg.Searcher, parentID uint32) []areaNode {
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

func findArea(s *gopkg.Searcher, address string) (gopkg.Info, bool) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return gopkg.Info{}, false
		}
		ip = ips[0]
	}
	info, err := s.Find(ip.String())
	if err != nil || info.Area.ID == 0 {
		return gopkg.Info{}, false
	}
	return info, true
}

// Lookup returns the area chain for address (host or host:port), leaf first then parents.
// Empty if the IP is unknown or the DB is not open.
func Lookup(address string) *AreaList {
	geoMu.Lock()
	s := searcher
	geoMu.Unlock()
	if s == nil {
		return &AreaList{}
	}
	info, ok := findArea(s, address)
	if !ok {
		return &AreaList{}
	}
	out := make([]*Area, 0, 8)
	for a := &info.Area; a != nil; a = a.Parent {
		parentID := int64(0)
		if a.Parent != nil {
			parentID = int64(a.Parent.ID)
		}
		out = append(out, &Area{
			ID:       int64(a.ID),
			ParentID: parentID,
			Name:     a.Name,
		})
	}
	return &AreaList{items: out}
}
