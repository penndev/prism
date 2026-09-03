package proxy

import (
	"desktop/internal"
	"desktop/internal/lang"
	"desktop/internal/route"
	"desktop/internal/tun"
	"errors"

	"github.com/penndev/prism/stack"
)

func (p *Proxy) closeTunDev() {
	route.RestoreDNS()
	if p.netstack != nil {
		p.netstack.Close()
		p.netstack = nil
	}
	if p.dev == nil {
		return
	}
	p.dev.Close()
	p.dev = nil
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "tun dev close success")
}

func (p *Proxy) startTunDev() error {
	if u := p.remoteURL.Load(); u == nil || u.Host == "" {
		return errors.New(lang.DefaultLang.T("proxy.tun.noNode"))
	}
	if !tunPermission() {
		return nil
	}
	p.closeTunDev()
	dev, err := tun.New(tun.Options{
		Name:   TUN_NAME,
		MTU:    TUN_MTU,
		Offset: TUN_OFFSET,
	})
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "tun.New: "+err.Error())
		return errors.New(lang.DefaultLang.T("proxy.tun.startFailed"))
	}
	netstack, err := stack.New(stack.Option{
		EndPoint: dev,
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			p.HandleConnect(f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			p.HandleConnect(f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})
	if err != nil {
		dev.Close()
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "stack.New: "+err.Error())
		return errors.New(lang.DefaultLang.T("proxy.tun.startFailed"))
	}
	p.dev = dev
	p.netstack = netstack
	if err := route.Start(route.Options{
		DevName:      dev.Name(),
		DevIP:        TUN_IP,
		DevIP6:       TUN_IP6,
		RouteAddress: Routes,
	}); err != nil {
		p.closeTunDev()
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "route.Start: "+err.Error())
		return errors.New(lang.DefaultLang.T("proxy.tun.startFailed"))
	}
	return nil
}
