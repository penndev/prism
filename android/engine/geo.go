package engine

import (
	"encoding/json"
	"net"

	"github.com/penndev/prism/ipregion"
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
	return ipregion.Open(path)
}

// DBStatus returns the last opened file's metadata. Path is empty if never opened.
func DBStatus() *DbStatus {
	st := ipregion.StatusOf()
	return &DbStatus{
		Exists:  st.Exists,
		Path:    st.Path,
		Size:    st.Size,
		Version: st.Version,
		Remark:  st.Remark,
		Areas:   int64(st.Areas),
		V4:      int64(st.V4),
		V6:      int64(st.V6),
	}
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
	src := ipregion.Areas(parentID)
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
// Empty if the IP is unknown or the DB is not open.
func Lookup(address string) *AreaList {
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
	info, err := ipregion.Find(ip.String())
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
