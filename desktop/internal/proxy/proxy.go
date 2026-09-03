package proxy

import (
	"desktop/internal"
	"desktop/internal/tun"
	"desktop/internal/web"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/penndev/prism/proxy"
	"github.com/penndev/prism/transport"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type Proxy struct {
	proxy.Server

	// 远程代理信息，用于检查心跳。
	// SetRemote 写、updateDialer 循环读，所以用原子指针。
	remoteURL atomic.Pointer[url.URL]
	// tun用
	dev *tun.Tun
	// tun 模式的 gvisor 协议栈，关 tun 时要一起关掉
	netstack   *stack.Stack
	readBytes  atomic.Int64
	writeBytes atomic.Int64
}

var localHandle = transport.Local()

func (p *Proxy) SetStart(host, user, pass string) error {
	if p.HandleConnect == nil {
		p.HandleConnect = p.handleConnectHook(localHandle, func(network, address string) {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, network+" "+address)
		})
	}

	go p.updateDialer()

	internal.App.Event.Emit(
		internal.AppConfig.LogTypeName_STATUS,
		"localServer://"+user+":"+pass+"@"+host,
	)

	// 配置未变化，保持当前服务
	if p.Addr == host && p.Username == user && p.Password == pass {
		return nil
	}

	p.Server.Close()
	// 在起 goroutine 之前赋值：放进 goroutine 里就会和上面那三个比较、
	// 以及 ListenAndServe 里读 Addr 形成竞态。
	p.Addr = host
	p.Username = user
	p.Password = pass
	go func() {
		if err := p.ListenAndServe(); err != nil {
			internal.App.Event.Emit(
				internal.AppConfig.LogTypeName_STATUS,
				"p.ListenAndServe error: "+err.Error(),
			)
		}
	}()

	return nil
}

func (p *Proxy) SetRemote(remote string) error {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		p.remoteURL.Store(nil)
		p.HandleConnect = p.handleConnectHook(localHandle, func(network, address string) {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, network+" "+address)
		})
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "SetRemote-> local")
		return nil
	}
	remoteURL, err := url.Parse(remote)
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "SetRemote error: "+err.Error())
		return err
	}
	if remoteURL.Host != "" {
		internal.App.Event.Emit(
			internal.AppConfig.LogTypeName_STATUS,
			"SetRemote-> "+remoteURL.Scheme+"://"+remoteURL.User.String()+"@"+remoteURL.Host,
		)
		handle, err := transport.FromURL(remoteURL)
		if err != nil {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "SetRemote error: "+err.Error())
			return err
		}
		p.remoteURL.Store(remoteURL)
		p.HandleConnect = p.handleConnectHook(handle, func(network, address string) {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, network+" "+address)
		})
		return nil
	}
	return nil
}

func (p *Proxy) SetMode(mode string) error {
	internal.App.Event.Emit(
		internal.AppConfig.LogTypeName_STATUS,
		"SetMode-> "+mode,
	)
	var err error
	switch mode {
	case "tun":
		err = p.startTunDev()
	default:
		p.closeTunDev()
	}
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, err.Error())
		return err
	}
	return err
}

func (p *Proxy) SetStop() {
	p.closeTunDev()
	p.Server.Close()
	internal.App.Event.Emit(
		internal.AppConfig.LogTypeName_STATUS,
		"local server close success",
	)
}

func (p *Proxy) TrafficBytes() (read uint64, write uint64) {
	return uint64(p.readBytes.Load()), uint64(p.writeBytes.Load())
}

func New() *Proxy {
	p := &Proxy{}
	p.HandlerFunc = web.Route
	return p
}
