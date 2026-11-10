package route

import (
	"cmp"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const fallbackDNS = "8.8.8.8"

// Egress 是当前物理出口。某个地址族为 nil 表示没有可用的默认路由。
type Egress struct {
	V4, V6 *net.Interface
	DNS    []string // 至少有一个元素
}

type routeCandidate struct {
	index  int
	metric uint64
	order  int // metric 相同时，越小越优先
}

type tunDev struct {
	name  string
	index int
}

var (
	current   atomic.Pointer[Egress]
	publishMu sync.Mutex
	hook      func(prev, next *Egress)
	dnsCursor atomic.Uint32

	wake      = make(chan struct{}, 1)
	watchMu   sync.Mutex
	watchStop func()

	tun atomic.Pointer[tunDev]
)

// Iface 返回对应地址族的出口，s 为 nil 时也安全。
func (s *Egress) Iface(ipv6 bool) *net.Interface {
	if s == nil {
		return nil
	}
	if ipv6 {
		return s.V6
	}
	return s.V4
}

func (s *Egress) String() string {
	label := func(i *net.Interface) string {
		if i == nil {
			return "none"
		}
		return i.Name + "/" + strconv.Itoa(i.Index)
	}
	return "egress v4 " + label(s.V4) + " v6 " + label(s.V6)
}

// Load 返回最近一次选出来的物理出口。还没刷新过时是 nil。
func Load() *Egress { return current.Load() }

// Watch 在路由或地址变化时重新选出口。fn 在每次发布时调用，prev 第一次为 nil。
// 再次调用只换回调。配对的停止是 Unwatch。
func Watch(fn func(prev, next *Egress)) {
	watchMu.Lock()
	publishMu.Lock()
	hook = fn
	publishMu.Unlock()
	if watchStop == nil {
		done := make(chan struct{})
		cancel := listenRouteChanges()
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
			cancel()
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

// SourceIP 取出口网卡上与目标同地址族的一个全局单播地址。
func SourceIP(dst netip.Addr) net.IP {
	iface := current.Load().Iface(dst.Is6())
	if iface == nil {
		return nil
	}
	if ip := globalAddr(iface, dst.Is6()); ip.IsValid() {
		return ip.AsSlice()
	}
	return nil
}

// NextDNS 按当前出口的 DNS 列表轮询一个服务器。
func NextDNS() string {
	s := current.Load()
	if s == nil || len(s.DNS) == 0 {
		return fallbackDNS
	}
	return s.DNS[int(dnsCursor.Add(1)-1)%len(s.DNS)]
}

func pickDefault(ipv6 bool) *net.Interface {
	cands := defaultRoutes(ipv6)
	slices.SortFunc(cands, func(a, b routeCandidate) int {
		return cmp.Or(cmp.Compare(a.metric, b.metric), cmp.Compare(a.order, b.order))
	})
	for _, c := range cands {
		iface, err := net.InterfaceByIndex(c.index)
		if err != nil ||
			iface.Flags&net.FlagUp == 0 ||
			iface.Flags&net.FlagLoopback != 0 ||
			isOurTun(iface) ||
			!globalAddr(iface, ipv6).IsValid() {
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
		if ip, ok := netip.AddrFromSlice(ipnet.IP); ok {
			ip = ip.Unmap()
			if ip.IsGlobalUnicast() && ip.Is6() == ipv6 {
				return ip
			}
		}
	}
	return netip.Addr{}
}

func isOurTun(iface *net.Interface) bool {
	if t := tun.Load(); t != nil && t.index != 0 && t.index == iface.Index {
		return true
	}
	name := tunName()
	return name != "" && strings.EqualFold(iface.Name, name)
}

func tunName() string {
	if t := tun.Load(); t != nil {
		return t.name
	}
	return ""
}

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

func signal() {
	select {
	case wake <- struct{}{}:
	default:
	}
}

func pollRouteChanges(done <-chan struct{}) {
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
}
