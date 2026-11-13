package route

import (
	"math"
	"slices"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

func defaultRoutes(ipv6 bool) []routeCandidate {
	family := uint16(windows.AF_INET)
	if ipv6 {
		family = uint16(windows.AF_INET6)
	}
	var table *windows.MibIpForwardTable2
	if err := windows.GetIpForwardTable2(family, &table); err != nil {
		return nil
	}
	defer windows.FreeMibTable(unsafe.Pointer(table))

	var out []routeCandidate
	for _, row := range table.Rows() {
		if row.DestinationPrefix.PrefixLength != 0 || row.Loopback != 0 {
			continue
		}
		out = append(out, routeCandidate{
			index:  int(row.InterfaceIndex),
			metric: routeMetric(row.Metric, row.InterfaceIndex, family),
			order:  int(row.InterfaceIndex), // 并列时用较小的 ifIndex
		})
	}
	return out
}

// routeMetric 返回路由的实际优先级：路由偏移 + 接口度量。
// 读不到接口度量时排到最后，避免被当成最优路由。
func routeMetric(offset, ifIndex uint32, family uint16) uint64 {
	row := windows.MibIpInterfaceRow{Family: family, InterfaceIndex: ifIndex}
	if err := windows.GetIpInterfaceEntry(&row); err != nil {
		status("GetIpInterfaceEntry if=%d: %v", ifIndex, err)
		return math.MaxUint32/2 + uint64(offset)
	}
	return uint64(offset) + uint64(row.Metric)
}

func interfaceDNS(index int) []string {
	var servers []string
	err := walkAdapters(
		windows.AF_UNSPEC,
		windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST,
		func(aa *windows.IpAdapterAddresses) bool {
			if int(aa.IfIndex) != index && int(aa.Ipv6IfIndex) != index {
				return true
			}
			for dns := aa.FirstDnsServerAddress; dns != nil; dns = dns.Next {
				ip := sockaddrIP(dns.Address)
				if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
					continue
				}
				s := ip.String()
				if !slices.Contains(servers, s) {
					servers = append(servers, s)
				}
			}
			return false
		},
	)
	if err != nil {
		return nil
	}
	return servers
}

var (
	notifyCB   uintptr
	notifyOnce sync.Once
)

func listenRouteChanges() (cancel func()) {
	notifyOnce.Do(func() {
		notifyCB = windows.NewCallback(func(_, _, _ uintptr) uintptr {
			signal()
			return 0
		})
	})
	var handles []windows.Handle
	var h windows.Handle
	for _, family := range []uint16{windows.AF_INET, windows.AF_INET6} {
		if windows.NotifyRouteChange2(family, notifyCB, nil, false, &h) == nil {
			handles = append(handles, h)
		}
	}
	if windows.NotifyUnicastIpAddressChange(windows.AF_UNSPEC, notifyCB, nil, false, &h) == nil {
		handles = append(handles, h)
	}
	if len(handles) == 0 {
		return startRoutePoll()
	}
	return func() {
		for _, h := range handles {
			windows.CancelMibChangeNotify2(h)
		}
	}
}
