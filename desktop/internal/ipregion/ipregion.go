// Package ipregion å°è£åºç¨å ipregion.db çæå¼ä¸æ¥è¯¢ï¼ä¾è§åé¡µä¸è¿æ¥åæµå±ç¨ã
// åºæä»¶çä¸è½½/ä¸ä¼ ç± web å±è´è´£åå¥ DBPath åè°ç¨ Resetã
package ipregion

import (
	"desktop/internal/storage"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	gopkg "github.com/penndev/gopkg/ipregion"
)

const DBName = "ipregion.db"

// Status åºæä»¶ç¶æã
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

func init() {
	storage.SetRuleAreaNames(Names)
}

func DBPath() (string, error) {
	dir, err := storage.AppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DBName), nil
}

// Reset å³é­å¹¶æ¸ç©ºç¼å­ç Searcherï¼åºæä»¶æ´æ°åè°ç¨ï¼ã
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	if searcher != nil {
		_ = searcher.Close()
		searcher = nil
	}
}

// Get è¿åå·²æå¼ç Searcherï¼æªæå¼æ¶ä¸º nilã
func Get() *gopkg.Searcher {
	mu.Lock()
	defer mu.Unlock()
	return searcher
}

// Open æå¼åºç¨éç½®ç®å½ä¸ç ipregion.dbï¼å¸¦ç¼å­ï¼ã
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
		return nil, fmt.Errorf("æªæ¾å° ipregion.dbï¼è¯·åä¸è½½æä¸ä¼ ")
	}
	s, err := gopkg.Open(path)
	if err != nil {
		return nil, fmt.Errorf("æå¼ ipregion.db å¤±è´¥: %w", err)
	}
	searcher = s
	return searcher, nil
}

// StatusOf è¿ååºç¶æã
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

// Find æ¥è¯¢ IP å°åä¿¡æ¯ã
func Find(ip string) (gopkg.Info, error) {
	s, err := Open()
	if err != nil {
		return gopkg.Info{}, err
	}
	return s.Find(ip)
}

// Names æå°å ID æ¥è¯¢åç§°ï¼æ¾ä¸å°ç ID ä¼è¢«è·³è¿ã
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

// InAreas å¤æ­ addressï¼host æ host:portï¼æ¯å¦å±äºç»å®å°å IDï¼å«ä¸çº§ï¼ã
// æ æ³è§£ææä¸å¨åè¡¨ä¸­æ¶è¿å falseã
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
