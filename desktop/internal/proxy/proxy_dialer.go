package proxy

import (
	"desktop/internal"
	"desktop/internal/route"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/penndev/prism/fakeip"
	"github.com/penndev/prism/transport/dialer"
)

type nicAddrs struct {
	v4 []net.IP
	v6 []net.IP
}

// physicalNics 按网卡收集已 up 的地址，排除 loopback 和本进程 TUN。
func physicalNics() []nicAddrs {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	tun := strings.ToLower(TUN_NAME)
	var nics []nicAddrs
	for _, iface := range ifaces {
		n := strings.ToLower(iface.Name)
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if tun != "" && (n == tun || strings.HasPrefix(n, tun)) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		var nic nicAddrs
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
				continue
			}
			if a, ok := netip.AddrFromSlice(ip); ok {
				a = a.Unmap()
				if TUN_IP.Contains(a) || TUN_IP6.Contains(a) {
					continue
				}
			}
			if ip4 := ip.To4(); ip4 != nil {
				nic.v4 = append(nic.v4, ip4)
			} else {
				nic.v6 = append(nic.v6, ip)
			}
		}
		if len(nic.v4) > 0 || len(nic.v6) > 0 {
			nics = append(nics, nic)
		}
	}
	return nics
}

func hasIP(list []net.IP, ip net.IP) bool {
	for _, c := range list {
		if ip != nil && c.Equal(ip) {
			return true
		}
	}
	return false
}

// probe 用候选源地址去连节点，返回第一个能通的。
func probe(cands []net.IP, target string) net.IP {
	host, _, _ := net.SplitHostPort(target)
	dest := net.ParseIP(host)
	for _, ip := range cands {
		if dest != nil && (dest.To4() == nil) != (ip.To4() == nil) {
			continue
		}
		c, err := (&net.Dialer{Timeout: 2 * time.Second, LocalAddr: &net.TCPAddr{IP: ip}}).Dial("tcp", target)
		if err == nil {
			c.Close()
			return ip
		}
	}
	return nil
}

var dialerStarted atomic.Bool

func remoteTarget(u *url.URL) string {
	if u == nil || u.Host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(u.Host); err == nil {
		return u.Host
	}
	if u.Hostname() != "" && u.Port() != "" {
		return net.JoinHostPort(u.Hostname(), u.Port())
	}
	return ""
}

// updateDialer 由 SetStart 后台启动。等到节点地址后，按「能连上节点」绑定物理网卡。
// 已绑定的地址还在就保持；休眠导致旧地址暂时消失且新地址探测失败时，不改绑（避免落到 Hyper-V Default Switch）。
func (p *Proxy) updateDialer() {
	// 这是个不会返回的常驻循环，所以不能用 sync.Once：
	// Once.Do 会让后来的调用一直阻塞等第一个返回，
	// 于是每次 SetStart 都在这里漏一个 goroutine。
	if !dialerStarted.CompareAndSwap(false, true) {
		return
	}
	for {
		target := remoteTarget(p.remoteURL.Load())
		if target == "" {
			time.Sleep(time.Second)
			continue
		}
		nics := physicalNics()
		if b, ok := dialer.TCPDialer.(*dialer.BoundDialer); ok {
			keep := false
			for _, nic := range nics {
				if hasIP(nic.v4, b.LocalIPv4) || hasIP(nic.v6, b.LocalIPv6) {
					keep = true
					break
				}
			}
			if keep {
				time.Sleep(5 * time.Second)
				continue
			}
		}
		var v4, v6 net.IP
		for _, nic := range nics {
			p4, p6 := probe(nic.v4, target), probe(nic.v6, target)
			if p4 == nil && p6 == nil {
				continue
			}
			v4, v6 = p4, p6
			if v4 == nil && len(nic.v4) > 0 {
				v4 = nic.v4[0]
			}
			if v6 == nil && len(nic.v6) > 0 {
				v6 = nic.v6[0]
			}
			break
		}
		if v4 == nil && v6 == nil {
			time.Sleep(30 * time.Second)
			continue
		}
		if internal.App != nil {
			internal.App.Event.Emit(
				internal.AppConfig.LogTypeName_STATUS,
				fmt.Sprintf("Select Device v4 %v v6 %v", v4, v6),
			)
		}
		d := &dialer.BoundDialer{LocalIPv4: v4, LocalIPv6: v6}
		dialer.TCPDialer, dialer.UDPDialer = d, d
		if dns := route.CurrentDNS(TUN_NAME); len(dns) > 0 {
			fakeip.SetUpstream(dns[0])
		}
		fakeip.SetHandleConnect(localHandle)
		time.Sleep(time.Second)
	}
}
