package proxy

import (
	"desktop/internal"
	"desktop/internal/ipregion"
	"desktop/internal/storage"
	"desktop/internal/tun"
	"desktop/internal/web"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/penndev/prism/proxy"
	"github.com/penndev/prism/transport"
)

type Proxy struct {
	proxy.Server

	// 远程代理信息，用于检查心跳。
	remoteURL *url.URL
	// tun用
	dev        *tun.Tun
	readBytes  uint64
	writeBytes uint64
}

var localHandle = transport.Local()

func (p *Proxy) SetStart(host, user, pass string) error {
	if p.HandleConnect == nil {
		p.HandleConnect = p.handleConnectHook(localHandle, func(network, address string) {
			internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "local -> "+network+" "+address)
		})
	}
	dialerOnce.Do(func() {
		go func() {
			// 循环设置检查心跳。设置出网网卡的IP。来应对网络变化。比如无线切有线
			// 检查应对的目标服务器是 p.remoteURL
			for {
				if p.remoteURL == nil { // 等待设置远程代理信息。
					time.Sleep(1 * time.Second)
					continue
				} else {
					p.updateDialer()
					time.Sleep(10 * time.Second)
				}
			}
		}()
	})

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
	handle, err := HandleConnect(p.remoteURL)
	p.HandleConnect = p.handleConnectHook(handle, func(network, address string) {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, p.remoteURL.Scheme+" -> "+network+" "+address)
	})
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "SetRemote error: "+err.Error())
		return err
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
	read = atomic.LoadUint64(&p.readBytes)
	write = atomic.LoadUint64(&p.writeBytes)
	return
}

// RuleStatus 主页面展示用的地域规则摘要。
type RuleStatus struct {
	Mode      string   `json:"mode"`
	AreaNames []string `json:"areaNames"`
}

func (p *Proxy) GetRuleStatus() RuleStatus {
	out := RuleStatus{Mode: "global", AreaNames: []string{}}
	st := storage.DefaultStorage
	if st == nil {
		return out
	}
	cfg, err := st.GetRuleConfig()
	if err != nil || cfg == nil {
		return out
	}
	switch cfg.Mode {
	case "proxy", "bypass":
		out.Mode = cfg.Mode
		out.AreaNames = ipregion.Names(cfg.AreaIDs)
	}
	return out
}

func New() *Proxy {
	p := &Proxy{}
	p.HandlerFunc = web.Route
	return p
}
