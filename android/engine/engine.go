//go:build linux

package engine

import (
	"errors"
	"net"
	"net/url"
	"sync"
	"syscall"

	"github.com/penndev/prism/stack"
	"github.com/penndev/prism/transport"
	"github.com/penndev/prism/transport/dialer"
	"golang.org/x/sys/unix"
	gvisorstack "gvisor.dev/gvisor/pkg/tcpip/stack"
)

var (
	mu         sync.Mutex
	ep         gvisorstack.LinkEndpoint
	tunFD      = -1
	started    bool
	origTCP    dialer.Dialer
	origUDP    dialer.Dialer
	handler    Handler
	counter    *byteCounter
	errProtect = errors.New("protect failed")
	errStarted = errors.New("already started")
	localH     = transport.Local()
)

// Start attaches gVisor to a TUN fd and forwards TCP/UDP using opt.Proxy.
// Handler.UseProxy decides direct vs proxy per destination.
// mtu <= 0 uses 1500. Handler.Protect must wrap VpnService.protect.
func Start(opt *Options) error {
	mu.Lock()
	defer mu.Unlock()
	if started {
		return errStarted
	}
	if opt == nil {
		return errors.New("options required")
	}
	h := opt.Handler
	if h == nil {
		return errors.New("handler required")
	}
	mtu := opt.MTU
	if mtu <= 0 {
		mtu = 1500
	}

	r, err := url.Parse(opt.Proxy)
	if err != nil {
		return err
	}
	proxyH, err := transport.FromURL(r)
	if err != nil {
		return err
	}

	endpoint, err := createTUN(int(opt.FD), uint32(mtu))
	if err != nil {
		return err
	}

	installProtect(h)
	ctr := newByteCounter(h)
	stack.New(stack.Option{
		EndPoint: endpoint,
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			relay(proxyH, localH, h, ctr, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			relay(proxyH, localH, h, ctr, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})
	ctr.start()

	ep = endpoint
	tunFD = int(opt.FD)
	handler = h
	counter = ctr
	started = true
	return nil
}

// Stop closes the TUN fd so gVisor dispatch exits.
func Stop() {
	mu.Lock()
	if !started {
		mu.Unlock()
		return
	}
	ctr := counter
	fd := tunFD
	endpoint := ep
	started = false
	tunFD = -1
	ep = nil
	counter = nil
	handler = nil
	mu.Unlock()

	if fd >= 0 {
		_ = unix.Close(fd)
	}
	if endpoint != nil {
		endpoint.Wait()
	}
	ctr.halt()

	mu.Lock()
	restoreProtect()
	mu.Unlock()
}

func installProtect(h Handler) {
	origTCP = dialer.TCPDialer
	origUDP = dialer.UDPDialer
	if h == nil {
		return
	}
	d := &net.Dialer{Control: protectControl(h)}
	dialer.TCPDialer = d
	dialer.UDPDialer = d
}

func restoreProtect() {
	if origTCP != nil {
		dialer.TCPDialer = origTCP
		origTCP = nil
	}
	if origUDP != nil {
		dialer.UDPDialer = origUDP
		origUDP = nil
	}
}

func protectControl(h Handler) func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		var fail error
		if err := c.Control(func(fd uintptr) {
			if h == nil || !h.Protect(int32(fd)) {
				fail = errProtect
			}
		}); err != nil {
			return err
		}
		return fail
	}
}
