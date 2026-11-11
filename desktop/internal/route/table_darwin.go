package route

import (
	"time"

	netroute "golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

func interfaceDNS(int) []string { return nil }

func defaultRoutes(ipv6 bool) []routeCandidate {
	af := unix.AF_INET
	if ipv6 {
		af = unix.AF_INET6
	}
	rib, err := netroute.FetchRIB(af, netroute.RIBTypeRoute, 0)
	if err != nil {
		return nil
	}
	msgs, err := netroute.ParseRIB(netroute.RIBTypeRoute, rib)
	if err != nil {
		return nil
	}

	const skip = unix.RTF_REJECT | unix.RTF_BLACKHOLE | unix.RTF_IFSCOPE
	var out []routeCandidate
	for _, msg := range msgs {
		rm, ok := msg.(*netroute.RouteMessage)
		// x/net 把 rt_msghdr 偏移 28 的 rtm_use 读成 errno。
		// 默认路由一旦被用过，整条 0/0 会被当成错误丢掉，出口变成 none。
		if !ok {
			continue
		}
		if rm.Flags&unix.RTF_UP == 0 || rm.Flags&unix.RTF_GATEWAY == 0 || rm.Flags&skip != 0 {
			continue
		}
		// ParseRIB 保证 Addrs 长度为 RTAX_MAX，缺失的项为 nil。
		if !zeroAddr(rm.Addrs[unix.RTAX_DST], ipv6) {
			continue
		}
		if mask := rm.Addrs[unix.RTAX_NETMASK]; mask != nil && !zeroAddr(mask, ipv6) {
			continue
		}
		// 无 scope 的默认路由按内核返回顺序用，不按 ifIndex 重排。
		out = append(out, routeCandidate{index: rm.Index, order: len(out)})
	}
	return out
}

func zeroAddr(a netroute.Addr, ipv6 bool) bool {
	switch a := a.(type) {
	case *netroute.Inet4Addr:
		return !ipv6 && a.IP == [4]byte{}
	case *netroute.Inet6Addr:
		return ipv6 && a.IP == [16]byte{}
	}
	return false
}

func listenRouteChanges() (cancel func()) {
	fd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, 0)
	if err != nil {
		done := make(chan struct{})
		go pollRouteChanges(done)
		return func() { close(done) }
	}
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 2048)
		for {
			if _, err := unix.Read(fd, buf); err != nil {
				select {
				case <-done:
					return
				default:
					time.Sleep(time.Second)
				}
				continue
			}
			signal()
		}
	}()
	return func() {
		close(done)
		unix.Close(fd)
	}
}
