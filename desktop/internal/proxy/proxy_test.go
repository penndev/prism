package proxy

import (
	"net"
	"testing"
	"time"
)

// 需要本机 1080 端口上有可用的 socks5 代理，没有就跳过。
// 原来这里无条件 t.Fail()，等于整个包的 go test 永远是红的。
func TestProxyPing(t *testing.T) {
	const addr = "127.0.0.1:1080"
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		t.Skipf("%s 上没有代理在监听，跳过", addr)
	}
	conn.Close()

	result := (&ProxyPing{}).TestServer("socks5://"+addr, "www.example.com:80")
	if !result.Success {
		t.Fatalf("TestServer failed: %s", result.Error)
	}
	t.Logf("latency %dms", result.Latency)
}
