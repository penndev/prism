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
	ipRegionOpenMu sync.Mutex
	// regionMu 保护 region。Lookup 全程持读锁，换库后旧句柄才能安全 Close。
	ipRegionMu       sync.RWMutex
	ipRegionSearcher *ipregion.Searcher
)

// DBStatus is the summary of the opened ipregion.db.
type DBStatus struct {
	Version string `json:"version"`
	Remark  string `json:"remark"`
	Areas   int64  `json:"areas"`
	V4      int64  `json:"v4"`
	V6      int64  `json:"v6"`
}

// AreaList is the area chain returned by Lookup, leaf first then parents.
type AreaList struct {
	ids []int64
}

func (list *AreaList) Len() int            { return len(list.ids) }
func (list *AreaList) Get(index int) int64 { return list.ids[index] }

// SetIpregionDB 用 gopkg 的文件模式打开 path，换掉当前库。
// Open 会把段表解压到 tmpDir 下的 ipregion-* 目录，内存只留索引。
// 主 App 和扩展是两个进程，各调一次，各有自己的临时目录。
func SetIpregionDB(dbPath, tempDir string) error {
	ipRegionOpenMu.Lock()
	defer ipRegionOpenMu.Unlock()

	// ipregion 用 os.MkdirTemp("", ...) 建目录。Android 上 Go 默认的 /data/local/tmp 应用没有写权限。
	if err := os.Setenv("TMPDIR", tempDir); err != nil {
		return err
	}
	ipRegionMu.RLock()
	firstOpen := ipRegionSearcher == nil
	ipRegionMu.RUnlock()
	// 进程被系统杀掉时来不及 Close，旧目录会一直留在磁盘上。本进程还没打开过库，剩下的目录都是残留。
	// 只删目录。同目录里的 ipregion-download.tmp 也匹配这个前缀，不能一起删。
	if firstOpen {
		stale, _ := filepath.Glob(filepath.Join(tempDir, "ipregion-*"))
		for _, name := range stale {
			info, err := os.Stat(name)
			if err != nil || !info.IsDir() {
				continue
			}
			_ = os.RemoveAll(name)
		}
	}

	searcher, err := ipregion.Open(dbPath, ipregion.ModeFile)
	if err != nil {
		return err
	}
	ipRegionMu.Lock()
	previousSearcher := ipRegionSearcher
	ipRegionSearcher = searcher
	ipRegionMu.Unlock()
	if previousSearcher != nil {
		_ = previousSearcher.Close()
	}
	return nil
}

// CloseIpregionDB 关掉当前库。没打开时无操作。
func CloseIpregionDB() {
	ipRegionMu.Lock()
	previousSearcher := ipRegionSearcher
	ipRegionSearcher = nil
	ipRegionMu.Unlock()
	if previousSearcher != nil {
		_ = previousSearcher.Close()
	}
}

// CheckIpregionDB 试开 path，确认是可用的库，不替换当前库。
func CheckIpregionDB(path string) error {
	searcher, err := ipregion.Open(path, ipregion.ModeFile)
	if err != nil {
		return err
	}
	return searcher.Close()
}

// IpregionStatus returns nil when no ipregion.db is opened.
func IpregionStatus() *DBStatus {
	ipRegionMu.RLock()
	defer ipRegionMu.RUnlock()
	if ipRegionSearcher == nil {
		return nil
	}
	metadata := ipRegionSearcher.Meta()
	return &DBStatus{
		Version: metadata.Version,
		Remark:  metadata.Remark,
		Areas:   int64(metadata.Areas),
		V4:      int64(metadata.V4),
		V6:      int64(metadata.V6),
	}
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
	ipRegionMu.RLock()
	defer ipRegionMu.RUnlock()
	if ipRegionSearcher == nil {
		return "[]"
	}
	var build func(parentID uint32) []areaNode
	build = func(parentID uint32) []areaNode {
		areas := ipRegionSearcher.Areas(parentID)
		nodes := make([]areaNode, 0, len(areas))
		for _, area := range areas {
			nodes = append(nodes, areaNode{ID: area.ID, ParentID: area.ParentID, Name: area.Name, Children: build(area.ID)})
		}
		return nodes
	}
	data, err := json.Marshal(build(0))
	if err != nil {
		return "[]"
	}
	return string(data)
}

// Lookup returns the area chain for address (host or host:port), leaf first then parents.
// Domain names are resolved with a 2s timeout. Empty list when not found.
func Lookup(address string) *AreaList {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		cancel()
		if err != nil || len(addresses) == 0 {
			return &AreaList{}
		}
		ip = addresses[0]
	}

	ipRegionMu.RLock()
	defer ipRegionMu.RUnlock()
	if ipRegionSearcher == nil {
		return &AreaList{}
	}
	info, err := ipRegionSearcher.FindAddr(ip)
	if err != nil || info.Area.ID == 0 {
		return &AreaList{}
	}
	var ids []int64
	for a := &info.Area; a != nil; a = a.Parent {
		ids = append(ids, int64(a.ID))
	}
	return &AreaList{ids: ids}
}
