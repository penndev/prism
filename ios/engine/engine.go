package engine

import (
	"errors"
	"net/url"
	"strings"
	"sync"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/stack"
	"github.com/penndev/prism/transport"
	gvisorstack "gvisor.dev/gvisor/pkg/tcpip/stack"
)

// Handler is implemented on the iOS side (app or Packet Tunnel).
// NeedFake is called from fakeip for each DNS name; Swift owns the domain list.
// UseProxy decides direct vs proxy and should emit the connection log
// (e.g. "proxy tcp 1.2.3.4:443"). Swift looks up the IP (leaf → parent)
// and matches saved area IDs.
// OnProxyRead / OnProxyWrite report proxy-path Read/Write byte counts.
// WritePacket delivers an outbound IP packet to packetFlow (empty in the
// in-process mock handler).
type Handler interface {
	OnLog(line string)
	NeedFake(name string) bool
	UseProxy(network, address string) bool
	OnProxyRead(n int64)
	OnProxyWrite(n int64)
	WritePacket(pkt []byte)
}

// Options is passed to Start. Proxy is a URL:
// socks5://user:pass@host:port (also socks5s / http / https).
// iOS 没有 TUN fd；包经 WritePacket / Handler.WritePacket 进出。
type Options struct {
	MTU      int32
	Proxy    string
	Upstream string // VPN 启动前的系统 DNS，给未 fake 的查询用
	Handler  Handler
}

var (
	mu         sync.Mutex
	ep         *pktun
	netstack   *gvisorstack.Stack
	started    bool
	errStarted = errors.New("already started")
	localH     = transport.Local()
)

// WritePacket 把 packetFlow 读到的 IP 包注入协议栈。未 Start 时丢掉。
func WritePacket(pkt []byte) {
	mu.Lock()
	t := ep
	mu.Unlock()
	if t == nil {
		return
	}
	t.input(pkt)
}

// Start 挂 gVisor channel 端点和协议栈。包由 WritePacket / Handler.WritePacket 进出。
// mtu <= 0 按 1500。
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

	upstream := opt.Upstream
	if strings.TrimSpace(upstream) == "" {
		upstream = fakeip.DefaultUpstream
	}
	fakeip.SetUpstream(upstream)
	fakeip.SetNeedFake(h.NeedFake)

	endpoint := newPktun(uint32(mtu), h.WritePacket)
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
		endpoint.Close()
		endpoint.Wait()
		fakeip.SetNeedFake(nil)
		return err
	}

	ep = endpoint
	netstack = s
	started = true
	return nil
}

// Stop 关掉协议栈和 channel 端点，等到 outbound 泵退出。
func Stop() {
	mu.Lock()
	if !started {
		mu.Unlock()
		return
	}
	endpoint := ep
	s := netstack
	started = false
	ep = nil
	netstack = nil
	fakeip.SetNeedFake(nil)
	mu.Unlock()

	if s != nil {
		s.Close()
	}
	if endpoint != nil {
		endpoint.Close()
		endpoint.Wait()
	}
	if s != nil {
		s.Wait()
	}
}
