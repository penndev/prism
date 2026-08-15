package route

import (
	"log"
	"net/netip"
)

type Options struct {
	DevName      string
	DevIP        netip.Prefix
	RouteAddress []netip.Prefix
}

func Start(options Options) error {
	// 设置路由静态IP与掩码位
	err := SetDevAddr(options.DevName, options.DevIP)
	if err != nil {
		return err
	}
	for _, item := range options.RouteAddress {
		if err := SetRouteAddr(item, options.DevIP.Addr().AsSlice()); err != nil {
			// route add 失败不应静默，否则在 mac 上排查会非常困难
			log.Println("SetRouteAddr failed:", err)
		}
	}
	return err
}
