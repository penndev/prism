package proxy

import (
	"net"
	"sync/atomic"
)

type countConn struct {
	net.Conn
	readBytes  *uint64
	writeBytes *uint64
}

func (c *countConn) Read(p []byte) (n int, err error) {
	n, err = c.Conn.Read(p)
	if n > 0 && c.readBytes != nil {
		atomic.AddUint64(c.readBytes, uint64(n))
	}
	return
}

func (c *countConn) Write(p []byte) (n int, err error) {
	n, err = c.Conn.Write(p)
	if n > 0 && c.writeBytes != nil {
		atomic.AddUint64(c.writeBytes, uint64(n))
	}
	return
}

func (p *Proxy) wrapConn(conn net.Conn) net.Conn {
	return &countConn{
		Conn:       conn,
		readBytes:  &p.readBytes,
		writeBytes: &p.writeBytes,
	}
}
