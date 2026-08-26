package dns

import (
	"net"
	"strings"
	"sync"
	"time"
)

var (
	upMu        sync.RWMutex
	upstream    = "223.5.5.5"
	dohUpstream = "https://8.8.8.8/dns-query"
)

const (
	udpTimeout   = 5 * time.Second
	dohTimeout   = 5 * time.Second
	defaultProxy = "127.0.0.1:1080"
)

// SetUpstream 设置 UDP 上游 DNS。
func SetUpstream(up string) {
	up = strings.TrimSpace(up)
	if up == "" {
		return
	}
	if host, _, err := net.SplitHostPort(up); err == nil {
		up = host
	}
	upMu.Lock()
	upstream = up
	upMu.Unlock()
}

func currentUpstream() string {
	upMu.RLock()
	defer upMu.RUnlock()
	return upstream
}
