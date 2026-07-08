package proxy

import (
	"bufio"
	"net"
)

type Conn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *Conn) Read(p []byte) (n int, err error) {
	return c.reader.Read(p)
}

func (c *Conn) Write(p []byte) (n int, err error) {
	return c.Conn.Write(p)
}

func (c *Conn) Peek(n int) ([]byte, error) {
	return c.reader.Peek(n)
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{
		Conn:   conn,
		reader: bufio.NewReader(conn),
	}
}
