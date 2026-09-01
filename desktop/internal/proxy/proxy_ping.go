package proxy

import (
	"net/url"
	"strings"

	"github.com/penndev/prism/pkg"
)

type ProxyPing struct{}

type ProxyPingResult struct {
	Latency int    `json:"latency"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// TestServer 经代理向 latencyTestHost 发起 HTTP GET，测量首字节响应延迟。
func (p *ProxyPing) TestServer(serverURL string, latencyTestHost string) ProxyPingResult {
	if strings.TrimSpace(latencyTestHost) == "" {
		return ProxyPingResult{Success: false, Error: "empty latency test host"}
	}
	r, err := url.Parse(serverURL)
	if err != nil {
		return ProxyPingResult{Success: false, Error: err.Error()}
	}
	d, err := pkg.Ping(r, latencyTestHost)
	if err != nil {
		return ProxyPingResult{Success: false, Error: err.Error()}
	}
	return ProxyPingResult{Success: true, Latency: int(d.Milliseconds())}
}
