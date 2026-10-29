package engine

import (
	"context"
	"encoding/json"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/penndev/gopkg/ipregion"
)

var (
	// openMu 让 SetIpregionDB 串行：清残留目录时不能有另一次 Open 正在往临时目录里解压。
	openMu sync.Mutex
	// regionMu 保护 region。Lookup 全程持读锁，换库后旧句柄才能安全 Close。
	regionMu sync.RWMutex
	region   *ipregion.Searcher
)

// DBStatus is the summary of the opened ipregion.db.
type DBStatus struct {
	Version string
	Areas   int64
}

// AreaList is the area chain returned by Lookup, leaf first then parents.
type AreaList struct {
	ids []int64
}

func (l *AreaList) Len() int        { return len(l.ids) }
func (l *AreaList) Get(i int) int64 { return l.ids[i] }

// SetIpregionDB 用 gopkg 的文件模式打开 path，换掉当前库。
// Open 会把段表解压到 tmpDir 下的 ipregion-* 目录，内存只留索引。
// 主 App 和扩展是两个进程，各调一次，各有自己的临时目录。
func SetIpregionDB(path, tmpDir string) error {
	openMu.Lock()
	defer openMu.Unlock()

	// ipregion 用 os.MkdirTemp("", ...) 建目录。Android 上 Go 默认的 /data/local/tmp 应用没有写权限。
	if err := os.Setenv("TMPDIR", tmpDir); err != nil {
		return err
	}
	regionMu.RLock()
	firstOpen := region == nil
	regionMu.RUnlock()
	// 进程被系统杀掉时来不及 Close，旧目录会一直留在磁盘上。本进程还没打开过库，剩下的目录都是残留。
	// 只删目录。同目录里的 ipregion-download.tmp 也匹配这个前缀，不能一起删。
	if firstOpen {
		stale, _ := filepath.Glob(filepath.Join(tmpDir, "ipregion-*"))
		for _, name := range stale {
			info, err := os.Stat(name)
			if err != nil || !info.IsDir() {
				continue
			}
			_ = os.RemoveAll(name)
		}
	}

	s, err := ipregion.Open(path, ipregion.ModeFile)
	if err != nil {
		return err
	}
	regionMu.Lock()
	old := region
	region = s
	regionMu.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

// IpregionStatus returns nil when no ipregion.db is opened.
func IpregionStatus() *DBStatus {
	regionMu.RLock()
	defer regionMu.RUnlock()
	if region == nil {
		return nil
	}
	m := region.Meta()
	return &DBStatus{Version: m.Version, Areas: int64(m.Areas)}
}

type areaNode struct {
	ID       uint32     `json:"id"`
	ParentID uint32     `json:"parent_id"`
	Name     string     `json:"name"`
	Children []areaNode `json:"children"`
}

// AreaTree returns the full area tree as JSON:
// [{"id":1,"parent_id":0,"name":"...","children":[...]}]. "[]" when no db is opened.
func AreaTree() string {
	regionMu.RLock()
	defer regionMu.RUnlock()
	if region == nil {
		return "[]"
	}
	var build func(parent uint32) []areaNode
	build = func(parent uint32) []areaNode {
		areas := region.Areas(parent)
		out := make([]areaNode, 0, len(areas))
		for _, a := range areas {
			out = append(out, areaNode{ID: a.ID, ParentID: a.ParentID, Name: a.Name, Children: build(a.ID)})
		}
		return out
	}
	b, err := json.Marshal(build(0))
	if err != nil {
		return "[]"
	}
	return string(b)
}

// Lookup returns the area chain for address (host or host:port), leaf first then parents.
// Domain names are resolved with a 2s timeout. Empty list when not found.
func Lookup(address string) *AreaList {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		cancel()
		if err != nil || len(ips) == 0 {
			return &AreaList{}
		}
		addr = ips[0]
	}

	regionMu.RLock()
	defer regionMu.RUnlock()
	if region == nil {
		return &AreaList{}
	}
	info, err := region.FindAddr(addr)
	if err != nil || info.Area.ID == 0 {
		return &AreaList{}
	}
	var ids []int64
	for a := &info.Area; a != nil; a = a.Parent {
		ids = append(ids, int64(a.ID))
	}
	return &AreaList{ids: ids}
}
