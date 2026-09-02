package proxy

import (
	"desktop/internal"
	"desktop/internal/tun"
	"desktop/internal/web"
	"net/url"
	"sync/atomic"

	"github.com/penndev/prism/proxy"
	"github.com/penndev/prism/transport"
)

type Proxy struct {
	proxy.Server

	// 远程代理信息，用于检查心跳。
	remoteURL *url.URL
	// tun用
	dev        *tun.Tun
	readBytes  atomic.Int64
	writeBytes atomic.Int64
}

var localHandle = transport.Local()

func (p *Proxy) SetStart(host, user, pass string) error {
	if p.HandleConnect == nil {
		p.HandleConnect = p.handleConnectHook(localHandle, func(network, address string) {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "local -> "+network+" "+address)
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
	go func() {
		p.Addr = host
		p.Username = user
		p.Password = pass
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
	var err error
	p.remoteURL, err = url.Parse(remote)
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, err.Error())
	}
	internal.App.Event.Emit(
		internal.AppConfig.LogTypeName_STATUS,
		"SetRemote-> "+p.remoteURL.Scheme+"://"+p.remoteURL.User.String()+"@"+p.remoteURL.Host,
	)
	handle, err := transport.FromURL(p.remoteURL)
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "SetRemote error: "+err.Error())
		return err
	}
	p.HandleConnect = p.handleConnectHook(handle, func(network, address string) {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, p.remoteURL.Scheme+" -> "+network+" "+address)
	})
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
