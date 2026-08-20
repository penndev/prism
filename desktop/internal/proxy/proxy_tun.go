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
	if p.dev == nil {
		return
	}
	p.dev.Close()
	p.dev = nil
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "tun dev close success")
}

func (p *Proxy) startTunDev() error {
	if p.remoteURL == nil || p.remoteURL.Host == "" {
		return errors.New(lang.DefaultLang.T("proxy.tun.noNode"))
	}
	if !tunPermission() {
		return nil
	}
	p.closeTunDev()
	var err error
	p.dev, err = tun.New(tun.Options{
		Name:   TUN_NAME,
		MTU:    TUN_MTU,
		Offset: TUN_OFFSET,
	})
	if err != nil {
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "tun.New: "+err.Error())
		return errors.New(lang.DefaultLang.T("proxy.tun.startFailed"))
	}
	stack.New(stack.Option{
		EndPoint: p.dev,
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			// internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "tun -> "+f.RemoteAddr.Network()+" "+f.RemoteAddr.String())
			p.HandleConnect(f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			// internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "tun -> "+f.RemoteAddr.Network()+" "+f.RemoteAddr.String())
			p.HandleConnect(f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
	})
	route.Start(route.Options{
		DevName:      p.dev.Name(),
		DevIP:        TUN_IP,
		DevIP6:       TUN_IP6,
		RouteAddress: Routes,
	})
	return nil
}
