package engine

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/penndev/prism/transport"
)

type byteCounter struct {
	up, down atomic.Int64
	h        Handler
	stop     chan struct{}
	done     chan struct{}
}

func newByteCounter(h Handler) *byteCounter {
	return &byteCounter{
		h:    h,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
}

func (c *byteCounter) start() {
	go func() {
		defer close(c.done)
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				c.flush()
			case <-c.stop:
				c.flush()
				return
			}
		}
	}()
}

func (c *byteCounter) halt() {
	if c == nil {
		return
	}
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
	<-c.done
}

func (c *byteCounter) flush() {
	if c.h == nil {
		return
	}
	up := c.up.Swap(0)
	down := c.down.Swap(0)
	if up > 0 {
		c.h.OnProxyRead(up)
	}
	if down > 0 {
		c.h.OnProxyWrite(down)
	}
}

type countConn struct {
	net.Conn
	c *byteCounter
}

func wrapConn(conn net.Conn, c *byteCounter) net.Conn {
	if conn == nil || c == nil {
		return conn
	}
	return &countConn{Conn: conn, c: c}
}

func (c *countConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.c.up.Add(int64(n))
	}
	return n, err
}

func (c *countConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.c.down.Add(int64(n))
	}
	return n, err
}

func relay(proxy, local transport.HandleConnect, h Handler, ctr *byteCounter, conn net.Conn, network, address string) {
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
	} else {
		c = wrapConn(conn, ctr)
	}
	if err := handle(c, network, address); err != nil {
		if h != nil {
			h.OnLog(network + " " + address + " " + err.Error())
		}
		_ = conn.Close()
	}
}
