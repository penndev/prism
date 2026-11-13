package route

import (
	"desktop/internal"
	"fmt"
	"net"
	"net/netip"
)

// tunDNS 写给系统的 DNS。它落在进 TUN 的路由里，也不进 fakeip 地址池。
const tunDNS = "198.18.0.1"

type Options struct {
	DevName      string
	DevIP        netip.Prefix
	DevIP6       netip.Prefix
	RouteAddress []netip.Prefix
}

func Start(options Options) error {
	// 先记下 TUN。装路由期间如果系统通知刷新，才不会把这张网卡当成物理出口。
	dev := &tunDev{name: options.DevName}
	if iface, err := net.InterfaceByName(options.DevName); err == nil {
		dev.index = iface.Index
	}
	tun.Store(dev)
	defer Refresh()

	if err := SetDevAddr(options.DevName, options.DevIP); err != nil {
		return err
	}
	ip6 := options.DevIP6
	if ip6.IsValid() {
		if err := SetDevAddr(options.DevName, ip6); err != nil {
			status("SetDevAddr ipv6 failed: %s", err)
			ip6 = netip.Prefix{}
		}
	}
	for _, prefix := range options.RouteAddress {
		gw := gatewayFor(prefix, options.DevIP, ip6)
		if gw == nil {
			continue
		}
		if err := SetRouteAddr(prefix, gw); err != nil {
			status("SetRouteAddr failed: %s", err)
			if prefix.Addr().Is6() {
				return err // IPv4 失败只记日志，IPv6 失败就停
			}
		}
	}
	setDevDNS(options.DevName)
	return nil
}

// Stop 恢复系统 DNS，再按物理网卡选出口。
func Stop() {
	RestoreDNS()
	tun.Store(nil)
	Refresh()
}

func gatewayFor(prefix, v4, v6 netip.Prefix) net.IP {
	switch {
	case prefix.Addr().Is4() && v4.IsValid():
		return v4.Addr().AsSlice()
	case prefix.Addr().Is6() && v6.IsValid():
		return v6.Addr().AsSlice()
	default:
		return nil
	}
}

func prefixMask(p netip.Prefix) net.IP {
	return net.IP(net.CIDRMask(p.Bits(), p.Addr().BitLen()))
}

func status(format string, args ...any) {
	internal.App.Event.Emit(internal.AppConfig.LogTypeName_STATUS, fmt.Sprintf(format, args...))
}
