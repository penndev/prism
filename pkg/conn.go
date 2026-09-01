package pkg

import (
	"net"
	"sync/atomic"
)

// WrapConn counts bytes on conn. Read increments read, Write increments write.
// Nil conn, read, or write is ignored (conn is returned as-is if nil).
func WrapConn(conn net.Conn, read, write *atomic.Int64) net.Conn {
	if conn == nil || (read == nil && write == nil) {
		return conn
	}
	return &countConn{Conn: conn, read: read, write: write}
}

type countConn struct {
	net.Conn
	read, write *atomic.Int64
}

func (c *countConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 && c.read != nil {
		c.read.Add(int64(n))
	}
	return n, err
}

func (c *countConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 && c.write != nil {
		c.write.Add(int64(n))
	}
	return n, err
}
