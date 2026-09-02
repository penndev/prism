package engine

import (
	"encoding/json"
	"net"
	"os"

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

var dbPath string
var searcher *ipregion.Searcher

// SetIpregionDB opens ipregion.db at path. Java downloads/copies the file first.
func SetIpregionDB(path string) error {
	s, err := ipregion.Open(path)
	if err != nil {
		return err
	}
	searcher = s
	dbPath = path
	return nil
}

func DBStatus() *DbStatus {
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
	raw, err := json.Marshal(buildAreaTree(0))
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func buildAreaTree(parentID uint32) []areaNode {
	if searcher == nil {
		return nil
	}
	src := searcher.Areas(parentID)
	out := make([]areaNode, 0, len(src))
	for _, a := range src {
		out = append(out, areaNode{
			ID:       a.ID,
			ParentID: a.ParentID,
			Name:     a.Name,
			Children: buildAreaTree(a.ID),
		})
	}
	return out
}

// Lookup returns the area chain for address (host or host:port), leaf first then parents.
func Lookup(address string) *AreaList {
	if searcher == nil {
		return &AreaList{}
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return &AreaList{}
		}
		ip = ips[0]
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
