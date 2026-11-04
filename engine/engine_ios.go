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
	lifecycleMu       sync.Mutex
	endpoint          *packetTunnel
	networkStack      *gvisorstack.Stack
	errAlreadyStarted = errors.New("already started")
	directConnect     = transport.Local()
)

// WritePacket 把一批包注入协议栈。batch 是重复的 uint32 大端长度 + 包内容。未 Start 时丢掉。
func WritePacket(batch []byte) {
	lifecycleMu.Lock()
	tunnel := endpoint
	lifecycleMu.Unlock()
	if tunnel == nil {
		return
	}
	for len(batch) >= 4 {
		packetLength := int(binary.BigEndian.Uint32(batch[:4]))
		batch = batch[4:]
		if packetLength <= 0 || packetLength > len(batch) {
			return
		}
		tunnel.injectPacket(batch[:packetLength])
		batch = batch[packetLength:]
	}
}

// Start 挂 gVisor channel 端点和协议栈。包由 WritePacket / Handler.WritePacket 进出。
// mtu <= 0 按 1500。
func Start(options *Options) error {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	if endpoint != nil {
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

	upstream := options.Upstream
	if strings.TrimSpace(upstream) == "" {
		upstream = fakeip.DefaultUpstream
	}
	fakeip.SetUpstream(upstream)
	fakeip.SetNeedFake(handler.NeedFake)

	linkEndpoint := newPacketTunnel(uint32(mtu), handler.WritePacket)
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
		linkEndpoint.Close()
		linkEndpoint.outboundWorkers.Wait()
		fakeip.SetNeedFake(nil)
		return err
	}
	// FIN_WAIT_2 留 5 秒，让手机把最后一段读走。设成 0 会马上拆连接，网页尾部会卡住重试。
	// TIME_WAIT 只留 1 秒。默认各 60 秒时，Stop 里的 stack.Wait 会跟着等，端点也一直占着内存。
	lingerTimeout := tcpip.TCPLingerTimeoutOption(5 * time.Second)
	timeWaitTimeout := tcpip.TCPTimeWaitTimeoutOption(time.Second)
	_ = forwardingStack.SetTransportProtocolOption(tcp.ProtocolNumber, &lingerTimeout)
	_ = forwardingStack.SetTransportProtocolOption(tcp.ProtocolNumber, &timeWaitTimeout)
	// 收发窗口自动放大。闲时停在 64KB，跑起来最多 1MB。
	// packetFlow 这一跳有几毫秒延迟，窗口不张开的话速度上不去。
	bufSize := tcpip.TCPReceiveBufferSizeRangeOption{Min: 4 << 10, Default: 64 << 10, Max: 1 << 20}
	sendSize := tcpip.TCPSendBufferSizeRangeOption{Min: 4 << 10, Default: 64 << 10, Max: 1 << 20}
	moderateReceiveBuffer := tcpip.TCPModerateReceiveBufferOption(true)
	_ = forwardingStack.SetTransportProtocolOption(tcp.ProtocolNumber, &bufSize)
	_ = forwardingStack.SetTransportProtocolOption(tcp.ProtocolNumber, &sendSize)
	_ = forwardingStack.SetTransportProtocolOption(tcp.ProtocolNumber, &moderateReceiveBuffer)

	// Packet Tunnel 扩展整个进程限 50MB，Swift 那边也要用。文件模式的 ipregion 索引也在 Go 堆里，给 36MB。
	debug.SetMemoryLimit(36 << 20)
	debug.SetGCPercent(50)

	endpoint = linkEndpoint
	networkStack = forwardingStack
	return nil
}

// Stop 关掉协议栈和 channel 端点，等到 outbound 泵退出再返回。
// 先把 endpoint 置空再解锁：WritePacket 也要这把锁，读回调卡在锁上时写包协程回不来。
func Stop() {
	lifecycleMu.Lock()
	tunnel := endpoint
	forwardingStack := networkStack
	if tunnel == nil {
		lifecycleMu.Unlock()
		return
	}
	endpoint = nil
	networkStack = nil
	fakeip.SetNeedFake(nil)
	lifecycleMu.Unlock()

	if forwardingStack != nil {
		forwardingStack.Close()
	}
	tunnel.Close()
	tunnel.outboundWorkers.Wait()
	if forwardingStack != nil {
		forwardingStack.Wait()
	}
}

// Goroutines 返回当前 goroutine 数，给扩展的内存打点用：一直涨说明有连接没收尾。
func Goroutines() int64 {
	return int64(runtime.NumGoroutine())
}
