package proxy

import (
	"desktop/internal"
	"desktop/internal/lang"
	"desktop/internal/network"
	"desktop/internal/route"
	"desktop/internal/tun"
	"errors"
	"net"
	"net/netip"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/stack"
)

func (p *Proxy) closeTunDev() {
	if p.netstack != nil {
		p.netstack.Close()
		p.netstack = nil
	}
	if p.dev == nil {
		return
	}
	p.dev.Close()
	p.dev = nil
	route.Stop()
	network.Stop()
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
		// 没有 IPv6 出口时，直连的 IPv6 在握手前拒绝，客户端才能退回 IPv4。代理和 DNS 仍接收。
		Refuse: func(network string, dst netip.Addr, port uint16) bool {
			if !dst.Is6() || route.Load().Iface(true) != nil {
				return false
			}
			if network == "udp" && port == 53 {
				return false
			}
			if _, ok := fakeip.Lookup(net.IP(dst.AsSlice())); ok {
				return false
			}
			return true
		},
		HandleTCP: func(f *stack.ForwarderTCPRequest) {
			p.HandleConnect(f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandlerUDP: func(f *stack.ForwarderUDPRequest) {
			p.HandleConnect(f.Conn, f.RemoteAddr.Network(), f.RemoteAddr.String())
		},
		HandleICMP: func(src, dst netip.Addr, packet []byte) {
			p.handleICMP(dev, src, dst, packet)
		},
	})
	if err != nil {
		dev.Close()
		internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, "stack.New: "+err.Error())
		return errors.New(lang.DefaultLang.T("proxy.tun.startFailed"))
	}
	p.dev = dev
	p.netstack = netstack
	// 先于改路由装上拨号器，这段时间里连节点的流量不会进 TUN。
	network.Start()
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
