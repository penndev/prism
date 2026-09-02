//go:build linux

package engine

import (
	"errors"
	"net"
	"net/url"
	"sync"
	"syscall"

	"github.com/penndev/prism/fakeip"
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
	errProtect = errors.New("protect failed")
	errStarted = errors.New("already started")
	localH     = transport.Local()
)

// Options is passed to Start. Proxy is a URL:
// socks5://user:pass@host:port (also socks5s / http / https).
type Options struct {
	FD       int32
	MTU      int32
	Proxy    string
	Upstream string // VPN 启动前的系统 DNS，给未 fake 的查询用
	Handler  Handler
}

// Handler is implemented on the Android side.
// Protect must wrap VpnService.protect.
// NeedFake is called from fakeip for each DNS name; Java owns the domain list.
// UseProxy decides direct vs proxy and should emit the connection log
// (e.g. "proxy tcp 1.2.3.4:443"). Java looks up the IP (leaf → parent)
// and matches saved area IDs.
// OnProxyRead / OnProxyWrite report proxy-path Read/Write byte counts.
type Handler interface {
	Protect(fd int32) bool
	OnLog(line string)
	NeedFake(name string) bool
	UseProxy(network, address string) bool
	OnProxyRead(n int64)
	OnProxyWrite(n int64)
}

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
	fakeip.SetUpstream(opt.Upstream)
	fakeip.SetNeedFake(func(name string) bool {
		return h.NeedFake(name)
	})
	stack.New(stack.Option{
		EndPoint: endpoint,
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			relay(proxyH, localH, h, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			relay(proxyH, localH, h, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})

	ep = endpoint
	tunFD = int(opt.FD)
	handler = h
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
	fd := tunFD
	endpoint := ep
	started = false
	tunFD = -1
	ep = nil
	handler = nil
	fakeip.SetNeedFake(nil)
	mu.Unlock()

	if fd >= 0 {
		_ = unix.Close(fd)
	}
	if endpoint != nil {
		endpoint.Wait()
	}

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
