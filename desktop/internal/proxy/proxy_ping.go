package proxy

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ProxyPing struct{}

type ProxyPingResult struct {
	Latency int    `json:"latency"` // 延迟（毫秒）
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// TestServer 通过 HandleConnect 经代理向 latencyTestHost 发起 HTTP GET，测量首字节响应延迟。
func (p *ProxyPing) TestServer(serverURL string, latencyTestHost string) ProxyPingResult {
	s := strings.TrimSpace(latencyTestHost)
	if s == "" {
		return ProxyPingResult{Success: false, Error: "empty latency test host"}
	}

	host, portStr, splitErr := net.SplitHostPort(s)
	if splitErr != nil {
		host = s
		portStr = "80"
	}
	if host == "" {
		return ProxyPingResult{Success: false, Error: "invalid latency test host"}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return ProxyPingResult{Success: false, Error: "invalid port"}
	}

	dialAddr := net.JoinHostPort(host, strconv.Itoa(port))
	hostHdr := host
	if port != 80 {
		hostHdr = net.JoinHostPort(host, strconv.Itoa(port))
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		(&url.URL{Scheme: "http", Host: dialAddr, Path: "/"}).String(), nil)
	if err != nil {
		return ProxyPingResult{Success: false, Error: err.Error()}
	}
	httpReq.Host = hostHdr
	httpReq.Header.Set("Connection", "close")
	httpReq.Header.Set("User-Agent", "Prism-Desktop/1.0")

	var buf bytes.Buffer
	if err := httpReq.Write(&buf); err != nil {
		return ProxyPingResult{Success: false, Error: err.Error()}
	}
	req := buf.Bytes()

	remote, err := url.Parse(serverURL)
	if err != nil {
		return ProxyPingResult{Success: false, Error: err.Error()}
	}
	handle, err := HandleConnect(remote)
	if err != nil {
		return ProxyPingResult{Success: false, Error: err.Error()}
	}

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	overCH := make(chan error, 1)
	go func() {
		overCH <- handle(c1, "tcp", dialAddr)
	}()

	start := time.Now()
	deadline := 5 * time.Second
	_ = c2.SetDeadline(start.Add(deadline))
	if _, err := c2.Write(req); err != nil {
		return ProxyPingResult{Success: false, Error: err.Error()}
	}
	go func() {
		buf := make([]byte, 1)
		_, err := c2.Read(buf)
		overCH <- err
	}()

	timer := time.NewTimer(deadline)
	defer timer.Stop()
	select {
	case err := <-overCH:
		if err != nil {
			return ProxyPingResult{Success: false, Error: err.Error()}
		}
		return ProxyPingResult{Success: true, Latency: int(time.Since(start).Milliseconds())}
	case <-timer.C:
		return ProxyPingResult{Success: false, Error: "timeout"}
	}
}
