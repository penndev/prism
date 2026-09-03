package internal

import (
	"sync"

	"github.com/penndev/gopkg/ipregion"
)

// ipregion 库句柄被 HTTP 处理器（换库、读元信息、列地区）和每条连接的地域判定同时访问，
// 必须加锁：换库时若把正在 Find 的句柄关掉就是 use-after-close。
var (
	searcherMu sync.RWMutex
	searcher   *ipregion.Searcher
)

// AcquireSearcher 取库句柄并持读锁，返回 nil 表示还没打开。
// 用完必须调 ReleaseSearcher；期间 SetSearcher 会阻塞。
func AcquireSearcher() *ipregion.Searcher {
	searcherMu.RLock()
	return searcher
}

func ReleaseSearcher() {
	searcherMu.RUnlock()
}

// SetSearcher 换上新句柄并关闭旧的。传 nil 表示只关闭。
func SetSearcher(s *ipregion.Searcher) {
	searcherMu.Lock()
	old := searcher
	searcher = s
	searcherMu.Unlock()
	if old != nil && old != s {
		_ = old.Close()
	}
}

// EnsureSearcher 库未打开时按 path 打开一次。
func EnsureSearcher(path string) error {
	searcherMu.Lock()
	defer searcherMu.Unlock()
	if searcher != nil {
		return nil
	}
	s, err := ipregion.Open(path)
	if err != nil {
		return err
	}
	searcher = s
	return nil
}
