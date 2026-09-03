//go:build linux

package engine

import (
	"fmt"
	"net"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/pkg"
	"github.com/penndev/prism/transport"
)

// relay 处理一条来自 tun 的连接。
// h 一定非 nil：Start 在 Handler 为空时就返回错误了。
func relay(proxy, local transport.HandleConnect, h Handler, conn net.Conn, network, address string) {
	defer func() {
		if r := recover(); r != nil {
			h.OnLog(fmt.Sprintf("relay panic %s %s: %v", network, address, r))
		}
	}()
	if conn == nil {
		return
	}

	if host, port, err := net.SplitHostPort(address); err == nil {
		// DNS 一律本地劫持，由 fakeip 决定返回假 IP 还是转发给上游
		if port == "53" && (network == "udp" || network == "udp4" || network == "udp6") {
			_ = fakeip.Handle(conn, network, address)
			return
		}
		// 目标是之前发出去的假 IP，换回真实域名再走代理
		if domain, ok := fakeip.Lookup(host); ok {
			fakeaddr := net.JoinHostPort(domain, port)
			h.OnLog("fakeip " + network + " " + address + " " + fakeaddr)
			proxyTo(proxy, h, conn, network, fakeaddr)
			return
		}
	}

	if h.UseProxy(network, address) {
		proxyTo(proxy, h, conn, network, address)
		return
	}
	// 直连不计入代理流量，所以不包 WrapConn
	if err := local(conn, network, address); err != nil {
		h.OnLog(network + " " + address + " " + err.Error())
		_ = conn.Close()
	}
}

// proxyTo 走代理路径：包上流量统计再转发，失败时关连接并上报。
func proxyTo(proxy transport.HandleConnect, h Handler, conn net.Conn, network, address string) {
	if err := proxy(pkg.WrapConn(conn, h.OnProxyRead, h.OnProxyWrite), network, address); err != nil {
		h.OnLog(network + " " + address + " " + err.Error())
		_ = conn.Close()
	}
}
