//go:build linux

package engine

import (
	"net"

	"github.com/penndev/prism/pkg"
	"github.com/penndev/prism/transport"
)

func relay(proxy, local transport.HandleConnect, h Handler, conn net.Conn, network, address string) {
	defer func() { recover() }()
	if conn == nil {
		return
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
