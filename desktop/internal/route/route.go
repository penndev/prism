package route

import (
	"log"
	"net"
	"net/netip"
)

type Options struct {
	DevName      string
	DevIP        netip.Prefix
	DevIP6       netip.Prefix
	RouteAddress []netip.Prefix
}

func gatewayFor(route, v4, v6 netip.Prefix) net.IP {
	if route.Addr().Is4() && v4.IsValid() {
		return v4.Addr().AsSlice()
	}
	if route.Addr().Is6() && v6.IsValid() {
		return v6.Addr().AsSlice()
	}
	return nil
}

func Start(options Options) error {
	err := SetDevAddr(options.DevName, options.DevIP)
	if err != nil {
		return err
	}
	ip6 := options.DevIP6
	if ip6.IsValid() {
		if err := SetDevAddr(options.DevName, ip6); err != nil {
			log.Println("SetDevAddr ipv6 failed:", err)
			ip6 = netip.Prefix{}
		}
	}
	for _, item := range options.RouteAddress {
		gw := gatewayFor(item, options.DevIP, ip6)
		if gw == nil {
			continue
		}
		if err := SetRouteAddr(item, gw); err != nil {
			log.Println("SetRouteAddr failed:", err)
		}
	}
	setDevDNS(options.DevName)
	return err
}
