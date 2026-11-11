package network

import (
	"desktop/internal/route"
	"net"
	"strings"
	"testing"
)

func TestEgress(t *testing.T) {
	route.Refresh()
	state := route.Load()
	if state == nil {
		t.Fatal("egress not set")
	}
	t.Log(state) // 走 String()

	if state.V4 != nil {
		conn, err := dialOutbound("udp", "1.1.1.1:53")
		if err != nil {
			t.Fatalf("udp: %v", err)
		}
		local := conn.LocalAddr().(*net.UDPAddr)
		conn.Close()
		if local.IP.Equal(net.ParseIP("172.19.0.1")) || local.IP.Equal(net.ParseIP("198.18.0.1")) {
			t.Fatalf("udp source is the tun address %s", local.IP)
		}
		t.Logf("udp local %s", local)
	}

	_, err := dialOutbound("tcp", "127.0.0.1:1")
	if err != nil && strings.Contains(err.Error(), "network unreachable") {
		t.Fatalf("loopback was bound to the physical interface: %v", err)
	}
}
