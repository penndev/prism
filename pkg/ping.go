package pkg

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/penndev/prism/transport"
)

const pingDeadline = 5 * time.Second

// Ping measures HTTP TTFB through the proxy in proxyURL.
// latencyHost is host or host:port (default port 80).
func Ping(proxyURL *url.URL, latencyHost string) (time.Duration, error) {
	host, portStr, splitErr := net.SplitHostPort(strings.TrimSpace(latencyHost))
	if splitErr != nil {
		host = strings.TrimSpace(latencyHost)
		portStr = "80"
	}
	if host == "" {
		return 0, fmt.Errorf("invalid latency test host")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid port")
	}

	dialAddr := net.JoinHostPort(host, strconv.Itoa(port))
	hostHdr := host
	if port != 80 {
		hostHdr = net.JoinHostPort(host, strconv.Itoa(port))
	}

	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		(&url.URL{Scheme: "http", Host: dialAddr, Path: "/"}).String(), nil)
	if err != nil {
		return 0, err
	}
	httpReq.Host = hostHdr
	httpReq.Header.Set("Connection", "close")
	httpReq.Header.Set("User-Agent", "Prism")

	var buf bytes.Buffer
	if err := httpReq.Write(&buf); err != nil {
		return 0, err
	}

	handle, err := transport.FromURL(proxyURL)
	if err != nil {
		return 0, err
	}

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	overCH := make(chan error, 2)
	go func() {
		overCH <- handle(c1, "tcp", dialAddr)
	}()

	start := time.Now()
	_ = c2.SetDeadline(start.Add(pingDeadline))
	if _, err := c2.Write(buf.Bytes()); err != nil {
		return 0, err
	}
	go func() {
		one := make([]byte, 1)
		_, err := c2.Read(one)
		overCH <- err
	}()

	timer := time.NewTimer(pingDeadline)
	defer timer.Stop()
	select {
	case err := <-overCH:
		if err != nil {
			return 0, err
		}
		return time.Since(start), nil
	case <-timer.C:
		return 0, fmt.Errorf("timeout")
	}
}
