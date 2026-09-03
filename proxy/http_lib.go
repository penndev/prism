package proxy

import (
	"fmt"
	"net"
)

// HTTP 代理头
var HttpProxyHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// 单连接 Listener
type HttpSingleConnListener struct {
	conn net.Conn
	// Accept 会把 conn 置空，Addr 仍要能返回地址。
	addr net.Addr
}

func (l *HttpSingleConnListener) Accept() (net.Conn, error) {
	if l.conn == nil {
		return nil, fmt.Errorf("closed")
	}
	c := l.conn
	l.addr = c.LocalAddr()
	l.conn = nil
	return c, nil
}

func (l *HttpSingleConnListener) Close() error {
	return nil
}

func (l *HttpSingleConnListener) Addr() net.Addr {
	if l.addr != nil {
		return l.addr
	}
	if l.conn != nil {
		return l.conn.LocalAddr()
	}
	return &net.TCPAddr{}
}
