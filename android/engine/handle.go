//go:build linux

package engine

import (
	"net"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/pkg"
	"github.com/penndev/prism/transport"
)

func relay(proxy, local transport.HandleConnect, h Handler, conn net.Conn, network, address string) {
	defer func() { recover() }()
	if conn == nil {
		return
	}

	host, port, err := net.SplitHostPort(address)
	if err == nil && (network == "udp" || network == "udp4" || network == "udp6") && port == "53" {
		_ = fakeip.Handle(conn, network, address)
		return
	}
	if err == nil {
		if domain, ok := fakeip.Lookup(host); ok {
			fakeaddr := net.JoinHostPort(domain, port)
			if h != nil {
				h.OnLog("fakeip " + network + " " + address + " " + fakeaddr)
			}
			c := conn
			if h != nil {
				c = pkg.WrapConn(conn, h.OnProxyRead, h.OnProxyWrite)
			}
			if err := proxy(c, network, fakeaddr); err != nil {
				if h != nil {
					h.OnLog(network + " " + fakeaddr + " " + err.Error())
				}
				_ = conn.Close()
			}
			return
		}
	}

	useProxy := true
	if h != nil {
		useProxy = h.UseProxy(network, address)
	}
	handle := proxy
	c := conn
	if !useProxy {
		handle = local
	} else if h != nil {
		c = pkg.WrapConn(conn, h.OnProxyRead, h.OnProxyWrite)
	}
	if err := handle(c, network, address); err != nil {
		if h != nil {
			h.OnLog(network + " " + address + " " + err.Error())
		}
		_ = conn.Close()
	}
}
