package pkg

import (
	"net"
)

// WrapConn reports Read/Write byte counts via onRead/onWrite. Nil callbacks are skipped.
func WrapConn(conn net.Conn, onRead, onWrite func(int64)) net.Conn {
	if conn == nil || (onRead == nil && onWrite == nil) {
		return conn
	}
	return &countConn{Conn: conn, onRead: onRead, onWrite: onWrite}
}

type countConn struct {
	net.Conn
	onRead, onWrite func(int64)
}

func (c *countConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 && c.onRead != nil {
		c.onRead(int64(n))
	}
	return n, err
}

func (c *countConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 && c.onWrite != nil {
		c.onWrite(int64(n))
	}
	return n, err
}
