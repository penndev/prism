//go:build android

package engine

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/stack"
	"github.com/penndev/prism/transport"
	"github.com/penndev/prism/transport/dialer"
	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/link/fdbased"
	gvisorstack "gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
)

var (
	lifecycleMu  sync.Mutex
	endpoint     gvisorstack.LinkEndpoint
	networkStack *gvisorstack.Stack
	tunFileDesc  = -1
	originalTCP  dialer.Dialer
	originalUDP  dialer.Dialer

	errProtectFailed  = errors.New("protect failed")
	errAlreadyStarted = errors.New("already started")
	directConnect     = transport.Local()
)

// Handler is implemented on the Android side.
// Protect must wrap VpnService.protect.
// NeedFake is called from fakeip for each DNS name; Java owns the domain list.
// UseProxy decides direct vs proxy. Kotlin writes the connection log there
// from network and address. OnConnection is the fake-IP path, which does not
// call UseProxy. OnStatus and OnDNS carry the other two logs. The app calls Lookup for the area chain
// (leaf → parent) and matches saved area IDs.
// OnProxyRead / OnProxyWrite report proxy-path Read/Write byte counts.
type Handler interface {
	Protect(fd int32) bool
	OnStatus(line string)
	OnConnection(network, address, err string)
	OnDNS(name, qtype, result string)
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

// Start attaches gVisor to a TUN fd and forwards TCP/UDP using options.Proxy.
// Handler.UseProxy decides direct vs proxy per destination.
// mtu <= 0 uses 1500. Handler.Protect must wrap VpnService.protect.
func Start(options *Options) error {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	if networkStack != nil {
		return errAlreadyStarted
	}
	if options == nil {
		return errors.New("options required")
	}
	handler := options.Handler
	if handler == nil {
		return errors.New("handler required")
	}
	mtu := options.MTU
	if mtu <= 0 {
		mtu = 1500
	}

	proxyURL, err := url.Parse(options.Proxy)
	if err != nil {
		return err
	}
	proxyConnect, err := transport.FromURL(proxyURL)
	if err != nil {
		return err
	}

	linkEndpoint, err := fdbased.New(&fdbased.Options{
		FDs: []int{int(options.FD)},
		MTU: uint32(mtu),
	})
	if err != nil {
		return err
	}

	originalTCP, originalUDP = dialer.TCPDialer, dialer.UDPDialer
	protectedDialer := &net.Dialer{Control: func(network, address string, rawConn syscall.RawConn) error {
		var protectErr error
		if err := rawConn.Control(func(fd uintptr) {
			if !handler.Protect(int32(fd)) {
				protectErr = errProtectFailed
			}
		}); err != nil {
			return err
		}
		return protectErr
	}}
	dialer.TCPDialer, dialer.UDPDialer = protectedDialer, protectedDialer

	// upstream 为空（比如 IPv6-only 网络下取不到 IPv4 DNS）时 SetUpstream 会早退，
	// 沿用上一次连接留下的值，所以这里显式回落到默认值。
	upstream := options.Upstream
	if strings.TrimSpace(upstream) == "" {
		upstream = fakeip.DefaultUpstream
	}
	fakeip.SetUpstream(upstream)
	fakeip.SetNeedFake(handler.NeedFake)

	forwardingStack, err := stack.New(stack.Option{
		EndPoint: linkEndpoint,
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			Relay(proxyConnect, directConnect, handler, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			Relay(proxyConnect, directConnect, handler, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})
	if err != nil {
		// stack.New 可能已经挂上 NIC 起了读循环，失败后 fd 会被 Kotlin 关掉，先停读循环。
		linkEndpoint.Attach(nil)
		fakeip.SetNeedFake(nil)
		dialer.TCPDialer, dialer.UDPDialer = originalTCP, originalUDP
		return err
	}
	// FIN_WAIT_2 留 5 秒，让对端把最后一段读走。设成 0 会马上拆连接，网页尾部会卡住重试。
	// TIME_WAIT 只留 1 秒。默认各 60 秒时，Stop 里的 stack.Wait 会跟着等。
	linger := tcpip.TCPLingerTimeoutOption(5 * time.Second)
	tw := tcpip.TCPTimeWaitTimeoutOption(time.Second)
	_ = forwardingStack.SetTransportProtocolOption(tcp.ProtocolNumber, &linger)
	_ = forwardingStack.SetTransportProtocolOption(tcp.ProtocolNumber, &tw)

	endpoint = linkEndpoint
	networkStack = forwardingStack
	tunFileDesc = int(options.FD)
	return nil
}

// Stop 当场掐掉协议栈里的全部连接，等收尾结束再返回。
// 不放到后台：连续开关时，下一次 Start 会和还在 Wait 的旧栈叠在一起。
func Stop() {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	if networkStack == nil {
		return
	}
	fakeip.SetNeedFake(nil)

	// Close 会 Abort 当前所有传输端点。读循环靠 eventfd 停，不能只关 tun fd。
	networkStack.Close()
	endpoint.Attach(nil)
	_ = unix.Close(tunFileDesc)
	networkStack.Wait()
	dialer.TCPDialer, dialer.UDPDialer = originalTCP, originalUDP

	endpoint = nil
	networkStack = nil
	tunFileDesc = -1
}
