package engine

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

// Ping measures TTFB in milliseconds through proxy (desktop-style URL).
// latencyHost is the HTTP target (host or host:port). Empty uses google.com.
// Returns a negative value on failure.
func Ping(proxy, latencyHost string) int64 {
	ms, err := ping(proxy, latencyHost)
	if err != nil {
		return -1
	}
	return ms
}

func ping(proxy, latencyHost string) (int64, error) {
	s := strings.TrimSpace(latencyHost)
	if s == "" {
		s = "google.com"
	}
	host, portStr, splitErr := net.SplitHostPort(s)
	if splitErr != nil {
		host = s
		portStr = "80"
	}
	if host == "" {
		return -1, pingError("invalid latency test host")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return -1, pingError("invalid port")
	}

	dialAddr := net.JoinHostPort(host, strconv.Itoa(port))
	hostHdr := host
	if port != 80 {
		hostHdr = net.JoinHostPort(host, strconv.Itoa(port))
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		(&url.URL{Scheme: "http", Host: dialAddr, Path: "/"}).String(), nil)
	if err != nil {
		return -1, err
	}
	httpReq.Host = hostHdr
	httpReq.Header.Set("Connection", "close")
	httpReq.Header.Set("User-Agent", "Prism-Android/0.1")

	var buf bytes.Buffer
	if err := httpReq.Write(&buf); err != nil {
		return -1, err
	}
	req := buf.Bytes()

	handle, err := selectHandle(proxy)
	if err != nil {
		return -1, err
	}

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	overCH := make(chan error, 2)
	go func() {
		overCH <- handle(c1, "tcp", dialAddr)
	}()

	start := time.Now()
	deadline := 5 * time.Second
	_ = c2.SetDeadline(start.Add(deadline))
	if _, err := c2.Write(req); err != nil {
		return -1, err
	}
	go func() {
		one := make([]byte, 1)
		_, err := c2.Read(one)
		overCH <- err
	}()

	timer := time.NewTimer(deadline)
	defer timer.Stop()
	select {
	case err := <-overCH:
		if err != nil {
			return -1, err
		}
		return time.Since(start).Milliseconds(), nil
	case <-timer.C:
		return -1, pingError("timeout")
	}
}

type pingError string

func (e pingError) Error() string { return string(e) }
