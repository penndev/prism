package engine

import (
	"net/url"
	"strings"

	"github.com/penndev/prism/pkg"
)

// PingCallback receives one latency result. ms < 0 means failure.
type PingCallback interface {
	OnResult(ms int64)
}

// PingAsync measures TTFB on a goroutine and calls back. It returns immediately.
func PingAsync(proxy, latencyHost string, cb PingCallback) {
	if cb == nil {
		return
	}
	go func() {
		cb.OnResult(Ping(proxy, latencyHost))
	}()
}

// Ping measures TTFB in milliseconds through proxy (desktop-style URL).
// latencyHost is the HTTP target (host or host:port). Empty uses google.com.
// Returns a negative value on failure.
func Ping(proxy, latencyHost string) int64 {
	host := strings.TrimSpace(latencyHost)
	if host == "" {
		return -1
	}
	r, err := url.Parse(proxy)
	if err != nil {
		return -1
	}
	d, err := pkg.Ping(r, host)
	if err != nil {
		return -1
	}
	return d.Milliseconds()
}
