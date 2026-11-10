package network

import (
	"desktop/internal"
	"desktop/internal/route"
	"net"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/transport/dialer"
)

// Start 在默认路由改到 TUN 之前调用，让本进程的出站走物理网卡。
func Start() {
	dialer.TCPDialer = Dialer{}
	dialer.UDPDialer = Dialer{}
	// logged 跟着这一次 Watch。每次启动都打当前出口，之后只有变了才再打。
	var logged string
	route.Watch(func(prev, next *route.Egress) {
		if prev == nil || prev.DNS[0] != next.DNS[0] {
			fakeip.SetUpstream(next.DNS[0])
			emit(internal.AppConfig.LogTypeName_DNS, "Upstream DNS "+next.DNS[0])
		}
		if line := next.String(); line != logged {
			logged = line
			emit(internal.AppConfig.LogTypeName_STATUS, line)
		}
	})
}

// Stop 在 TUN 设备和它的路由已经撤掉之后调用，拨号恢复为系统默认。
func Stop() {
	route.Unwatch()
	dialer.TCPDialer = &net.Dialer{}
	dialer.UDPDialer = &net.Dialer{}
}

func emit(event, msg string) {
	if internal.App != nil {
		internal.App.Event.Emit(event, msg)
	}
}
