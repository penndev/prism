//go:build ios

package engine

import (
	"encoding/binary"
	"errors"
	"net/url"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/stack"
	"github.com/penndev/prism/transport"
	"gvisor.dev/gvisor/pkg/tcpip"
	gvisorstack "gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
)

// Handler is implemented on the iOS side (app or Packet Tunnel).
// NeedFake is called from fakeip for each DNS name; Swift owns the domain list.
// UseProxy decides direct vs proxy. Swift writes the connection log there
// from network and address. OnConnection is the fake-IP path, which does not
// call UseProxy. OnStatus and OnDNS carry the other two logs. Swift calls Lookup for the area chain
// (leaf → parent) and matches saved area IDs.
// OnProxyRead / OnProxyWrite report proxy-path Read/Write byte counts.
// WritePacket delivers outbound packets to packetFlow.
// The bytes are a batch: repeated uint32-be length + packet bytes.
type Handler interface {
	OnStatus(line string)
	OnConnection(network, address, err string)
	OnDNS(name, qtype, result string)
	NeedFake(name string) bool
	UseProxy(network, address string) bool
	OnProxyRead(n int64)
	OnProxyWrite(n int64)
	WritePacket(blob []byte)
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
	errStarted = errors.New("already started")
	localH     = transport.Local()
)

// WritePacket 把一批包注入协议栈。blob 是重复的 uint32 大端长度 + 包内容。未 Start 时丢掉。
func WritePacket(blob []byte) {
	mu.Lock()
	t := ep
	mu.Unlock()
	if t == nil {
		return
	}
	for len(blob) >= 4 {
		n := int(binary.BigEndian.Uint32(blob[:4]))
		blob = blob[4:]
		if n <= 0 || n > len(blob) {
			return
		}
		t.input(blob[:n])
		blob = blob[n:]
	}
}

// Start 挂 gVisor channel 端点和协议栈。包由 WritePacket / Handler.WritePacket 进出。
// mtu <= 0 按 1500。
func Start(opt *Options) error {
	mu.Lock()
	defer mu.Unlock()
	if ep != nil {
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
			Relay(proxyH, localH, h, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			Relay(proxyH, localH, h, f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})
	if err != nil {
		endpoint.Close()
		endpoint.pumps.Wait()
		fakeip.SetNeedFake(nil)
		return err
	}
	// FIN_WAIT_2 留 5 秒，让手机把最后一段读走。设成 0 会马上拆连接，网页尾部会卡住重试。
	// TIME_WAIT 只留 1 秒。默认各 60 秒时，Stop 里的 stack.Wait 会跟着等，端点也一直占着内存。
	linger := tcpip.TCPLingerTimeoutOption(5 * time.Second)
	tw := tcpip.TCPTimeWaitTimeoutOption(time.Second)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &linger)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &tw)
	// 收发窗口自动放大。闲时停在 64KB，跑起来最多 1MB。
	// packetFlow 这一跳有几毫秒延迟，窗口不张开的话速度上不去。
	bufSize := tcpip.TCPReceiveBufferSizeRangeOption{Min: 4 << 10, Default: 64 << 10, Max: 1 << 20}
	sendSize := tcpip.TCPSendBufferSizeRangeOption{Min: 4 << 10, Default: 64 << 10, Max: 1 << 20}
	moderate := tcpip.TCPModerateReceiveBufferOption(true)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &bufSize)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &sendSize)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &moderate)

	// Packet Tunnel 扩展整个进程限 50MB，Swift 那边也要用。文件模式的 ipregion 索引也在 Go 堆里，给 36MB。
	debug.SetMemoryLimit(36 << 20)
	debug.SetGCPercent(50)

	ep = endpoint
	netstack = s
	return nil
}

// Stop 关掉协议栈和 channel 端点，等到 outbound 泵退出再返回。
// 先把 ep 置空再解锁：WritePacket 也要这把锁，读回调卡在锁上时 writePackets 回不来。
func Stop() {
	mu.Lock()
	endpoint := ep
	s := netstack
	if endpoint == nil {
		mu.Unlock()
		return
	}
	ep = nil
	netstack = nil
	fakeip.SetNeedFake(nil)
	mu.Unlock()

	if s != nil {
		s.Close()
	}
	endpoint.Close()
	endpoint.pumps.Wait()
	if s != nil {
		s.Wait()
	}
}

// Goroutines 返回当前 goroutine 数，给扩展的内存打点用：一直涨说明有连接没收尾。
func Goroutines() int64 {
	return int64(runtime.NumGoroutine())
}
