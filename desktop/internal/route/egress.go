package route

import (
	"cmp"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const fallbackDNS = "8.8.8.8"

// Egress 是当前物理出口。某个地址族为 nil 表示没有可用的默认路由。
type Egress struct {
	V4, V6 *net.Interface
	DNS    []string // Refresh 之后至少有一个
}

// 默认路由候选。metric 小的优先，相同则 order 小的优先。
type routeCandidate struct {
	index  int
	metric uint64
	order  int
}

// tun 是我们自己的网卡。选物理出口时要跳过它。
type tunDev struct {
	name  string
	index int
}

var (
	current   atomic.Pointer[Egress]
	dnsCursor atomic.Uint32
	tun       atomic.Pointer[tunDev]

	publishMu sync.Mutex
	hook      func(prev, next *Egress)
	watchMu   sync.Mutex
	watchStop func()
	wake      = make(chan struct{}, 1)
)

// Iface 返回对应地址族的出口。e 为 nil 时也安全。
func (e *Egress) Iface(ipv6 bool) *net.Interface {
	if e == nil {
		return nil
	}
	if ipv6 {
		return e.V6
	}
	return e.V4
}

func (e *Egress) String() string {
	switch {
	case e.V4 == nil && e.V6 == nil:
		return "egress none"
	case e.V6 == nil:
		return "egress " + e.V4.Name
	case e.V4 == nil:
		return "egress " + e.V6.Name + " v6"
	case e.V4.Index == e.V6.Index:
		return "egress " + e.V4.Name
	default:
		return "egress v4 " + e.V4.Name + " v6 " + e.V6.Name
	}
}

// Load 返回最近一次选出来的物理出口。还没刷新过时是 nil。
func Load() *Egress { return current.Load() }

// Watch 在路由或地址变化时重新选出口。fn 在每次发布时调用，第一次 prev 为 nil。
// 再次调用只换回调。配对的停止是 Unwatch。
func Watch(fn func(prev, next *Egress)) {
	watchMu.Lock()
	publishMu.Lock()
	hook = fn
	publishMu.Unlock()
	if watchStop == nil {
		done := make(chan struct{})
		stopListen := listenRouteChanges()
		go func() {
			for {
				select {
				case <-done:
					return
				case <-wake:
					time.Sleep(300 * time.Millisecond) // 合并短时间内的连续通知
					for len(wake) > 0 {
						<-wake
					}
					select {
					case <-done:
						return
					default:
						Refresh()
					}
				}
			}
		}()
		watchStop = func() {
			close(done)
			stopListen()
		}
	}
	watchMu.Unlock()
	Refresh()
}

// Unwatch 停止路由变化通知。之后 Refresh 仍会更新出口，但不再调用回调。
func Unwatch() {
	watchMu.Lock()
	stop := watchStop
	watchStop = nil
	watchMu.Unlock()
	if stop != nil {
		stop()
	}
	publishMu.Lock()
	hook = nil
	publishMu.Unlock()
}

// Refresh 按当前路由表重新选择出口，并通知 Watch 注册的回调。
func Refresh() {
	next := &Egress{V4: pickDefault(false), V6: pickDefault(true)}
	next.DNS = egressDNS(next.V4, next.V6)

	publishMu.Lock()
	defer publishMu.Unlock()
	prev := current.Swap(next)
	if hook != nil {
		hook(prev, next)
	}
}

func signal() {
	select {
	case wake <- struct{}{}:
	default:
	}
}

func startRoutePoll() func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				signal()
			}
		}
	}()
	return func() { close(done) }
}

// SourceIP 取出口网卡上与目标同地址族的一个全局单播地址。
func SourceIP(dst netip.Addr) net.IP {
	iface := current.Load().Iface(dst.Is6())
	if iface == nil {
		return nil
	}
	ip := globalAddr(iface, dst.Is6())
	if !ip.IsValid() {
		return nil
	}
	return ip.AsSlice()
}

// NextDNS 按当前出口的 DNS 列表轮询一个服务器。
func NextDNS() string {
	e := current.Load()
	if e == nil {
		return fallbackDNS
	}
	i := int(dnsCursor.Add(1)-1) % len(e.DNS)
	return e.DNS[i]
}

func pickDefault(ipv6 bool) *net.Interface {
	cands := defaultRoutes(ipv6)
	slices.SortFunc(cands, func(a, b routeCandidate) int {
		return cmp.Or(cmp.Compare(a.metric, b.metric), cmp.Compare(a.order, b.order))
	})
	for _, c := range cands {
		iface, err := net.InterfaceByIndex(c.index)
		if err != nil {
			continue
		}
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isOurTun(iface) || !globalAddr(iface, ipv6).IsValid() {
			continue
		}
		return iface
	}
	return nil
}

func globalAddr(iface *net.Interface, ipv6 bool) netip.Addr {
	addrs, _ := iface.Addrs()
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip, ok := netip.AddrFromSlice(ipnet.IP)
		if !ok {
			continue
		}
		ip = ip.Unmap()
		if ip.IsGlobalUnicast() && ip.Is6() == ipv6 {
			return ip
		}
	}
	return netip.Addr{}
}

func isOurTun(iface *net.Interface) bool {
	dev := tun.Load()
	if dev == nil {
		return false
	}
	if dev.index != 0 && dev.index == iface.Index {
		return true
	}
	return dev.name != "" && strings.EqualFold(iface.Name, dev.name)
}

func tunName() string {
	dev := tun.Load()
	if dev == nil {
		return ""
	}
	return dev.name
}

// 物理网卡上的 DNS 优先，没有再用系统当前的，最后退回 fallbackDNS。
func egressDNS(v4, v6 *net.Interface) []string {
	for _, iface := range []*net.Interface{v4, v6} {
		if iface == nil {
			continue
		}
		if servers := interfaceDNS(iface.Index); len(servers) > 0 {
			return servers
		}
	}
	if servers := CurrentDNS(tunName()); len(servers) > 0 {
		return servers
	}
	return []string{fallbackDNS}
}
