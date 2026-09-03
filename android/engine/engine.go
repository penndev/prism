//go:build linux

package engine

import (
	"errors"
	"net"
	"net/url"
	"strings"
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
	mu       sync.Mutex
	ep       gvisorstack.LinkEndpoint
	netstack *gvisorstack.Stack
	tunFD    = -1
	started  bool
	origTCP  dialer.Dialer
	origUDP  dialer.Dialer
	handler  Handler

	errProtect = errors.New("protect failed")
	errStarted = errors.New("already started")
	localH     = transport.Local()
)

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

// Options is passed to Start. Proxy is a URL:
// socks5://user:pass@host:port (also socks5s / http / https).
type Options struct {
	FD       int32
	MTU      int32
	Proxy    string
	Upstream string // VPN 启动前的系统 DNS，给未 fake 的查询用
	Handler  Handler
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
	// upstream 为空（比如 IPv6-only 网络下取不到 IPv4 DNS）时 SetUpstream 会早退，
	// 沿用上一次连接留下的值，所以这里显式回落到默认值。
	upstream := opt.Upstream
	if strings.TrimSpace(upstream) == "" {
		upstream = fakeip.DefaultUpstream
	}
	fakeip.SetUpstream(upstream)
	fakeip.SetNeedFake(h.NeedFake)

	s, err := stack.New(stack.Option{
		EndPoint: endpoint,
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			relay(proxyH, localH, h, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			relay(proxyH, localH, h, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})
	if err != nil {
		fakeip.SetNeedFake(nil)
		restoreProtect()
		return err
	}

	ep = endpoint
	netstack = s
	tunFD = int(opt.FD)
	handler = h
	started = true
	return nil
}

// Stop 关掉协议栈和 TUN fd。会阻塞到 gVisor 的 dispatch goroutine 全部退出，
// 调用方不要放在 Android 主线程上。
func Stop() {
	mu.Lock()
	if !started {
		mu.Unlock()
		return
	}
	fd := tunFD
	endpoint := ep
	s := netstack
	started = false
	tunFD = -1
	ep = nil
	netstack = nil
	handler = nil
	fakeip.SetNeedFake(nil)
	mu.Unlock()

	// 先停协议栈，不再派发新连接
	if s != nil {
		s.Close()
	}
	// fdbased 的 LinkEndpoint.Close() 是空实现，只能靠关 fd 把阻塞在 readv 的
	// dispatcher 唤醒，所以顺序必须是「关 fd -> Wait」。
	if fd >= 0 {
		_ = unix.Close(fd)
	}
	if endpoint != nil {
		endpoint.Wait()
	}
	if s != nil {
		s.Wait()
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
