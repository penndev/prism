package proxy

import (
	"net"

	"github.com/penndev/prism/pkg"
)

func (p *Proxy) wrapConn(conn net.Conn) net.Conn {
	return pkg.WrapConn(conn, &p.readBytes, &p.writeBytes)
}
