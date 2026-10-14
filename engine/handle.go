package engine

import (
	"fmt"
	"net"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/pkg"
	"github.com/penndev/prism/transport"
)

// Hook 是单条连接的业务回调。Android / iOS 的 Handler 已经包含这些方法。
type Hook interface {
	OnLog(line string)
	UseProxy(network, address string) bool
	OnProxyRead(n int64)
	OnProxyWrite(n int64)
}

// Relay 处理一条连接：DNS 交给 fakeip，假 IP 换回域名走代理，其余问 h.UseProxy。
// h 一定非 nil。失败时关掉 conn 并返回错误；成功路径不关 conn。
func Relay(proxy, local transport.HandleConnect, h Hook, conn net.Conn, network, address string) error {
	defer func() {
		if r := recover(); r != nil {
			h.OnLog(fmt.Sprintf("relay panic %s %s: %v", network, address, r))
		}
	}()
	if conn == nil {
		return nil
	}

	if host, port, err := net.SplitHostPort(address); err == nil {
		// DNS 一律本地劫持，由 fakeip 决定返回假 IP 还是转发给上游。
		// Handle 在读结束时返回 EOF，这是正常收尾，不往外传。
		if port == "53" && (network == "udp" || network == "udp4" || network == "udp6") {
			_ = fakeip.Handle(conn, network, address)
			return nil
		}
		// 目标是之前发出去的假 IP，换回真实域名再走代理。
		if domain, ok := fakeip.Lookup(host); ok {
			fakeaddr := net.JoinHostPort(domain, port)
			h.OnLog("fakeip " + network + " " + address + " " + fakeaddr)
			return proxyTo(proxy, h, conn, network, fakeaddr)
		}
	}

	if h.UseProxy(network, address) {
		return proxyTo(proxy, h, conn, network, address)
	}
	// 直连不计入代理流量，所以不包 WrapConn。
	if err := local(conn, network, address); err != nil {
		h.OnLog(network + " " + address + " " + err.Error())
		_ = conn.Close()
		return err
	}
	return nil
}

// proxyTo 走代理路径：包上流量统计再转发，失败时关连接并上报。
func proxyTo(proxy transport.HandleConnect, h Hook, conn net.Conn, network, address string) error {
	if err := proxy(pkg.WrapConn(conn, h.OnProxyRead, h.OnProxyWrite), network, address); err != nil {
		h.OnLog(network + " " + address + " " + err.Error())
		_ = conn.Close()
		return err
	}
	return nil
}
