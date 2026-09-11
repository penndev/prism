package engine

import (
	"errors"
	"net/url"
	"sync"

	"github.com/penndev/prism/transport"
)

// Handler is implemented on the iOS side (app or Packet Tunnel).
// NeedFake is called from fakeip for each DNS name; Swift owns the domain list.
// UseProxy decides direct vs proxy and should emit the connection log
// (e.g. "proxy tcp 1.2.3.4:443"). Swift looks up the IP (leaf → parent)
// and matches saved area IDs.
// OnProxyRead / OnProxyWrite report proxy-path Read/Write byte counts.
type Handler interface {
	OnLog(line string)
	NeedFake(name string) bool
	UseProxy(network, address string) bool
	OnProxyRead(n int64)
	OnProxyWrite(n int64)
}

// Options is passed to Start. Proxy is a URL:
// socks5://user:pass@host:port (also socks5s / http / https).
// iOS Network Extension 用 packetFlow 而不是 Android 那样的 TUN fd，
// 所以这里没有 FD；真正接 TUN 时再加。
type Options struct {
	MTU      int32
	Proxy    string
	Upstream string // VPN 启动前的系统 DNS，给未 fake 的查询用
	Handler  Handler
}

var (
	mu         sync.Mutex
	started    bool
	handler    Handler
	errStarted = errors.New("already started")
)

// Start 目前只校验代理 URL 并标记已启动。
// iOS 的 TUN 走 Network Extension 的 packetFlow，和 Android fd 不同，
// 协议栈还没接上；接上之后这里再挂 gVisor / fakeip。
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
	r, err := url.Parse(opt.Proxy)
	if err != nil {
		return err
	}
	if _, err := transport.FromURL(r); err != nil {
		return err
	}
	handler = h
	started = true
	h.OnLog("engine mock started (packet tunnel not wired yet)")
	return nil
}

// Stop 关掉模拟会话。接上协议栈之后要在这里等 dispatch 退出。
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	if !started {
		return
	}
	started = false
	handler = nil
}
