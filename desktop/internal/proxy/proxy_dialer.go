package proxy

import (
	"log"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/penndev/prism/transport/dialer"
)

// physicalIPs 收集已 up 的网卡地址，只排除 loopback 和本进程 TUN。
func physicalIPs() (v4, v6 []net.IP) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	tun := strings.ToLower(TUN_NAME)
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
				v4 = append(v4, ip4)
			} else {
				v6 = append(v6, ip)
			}
		}
	}
	return
}

// hasIP 判断 ip 是否还在网卡地址列表里。
func hasIP(list []net.IP, ip net.IP) bool {
	for _, c := range list {
		if ip != nil && c.Equal(ip) {
			return true
		}
	}
	return false
}

// probe 用候选源地址去连节点，返回第一个能通的；全失败返回 nil，不随便兜底。
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

var dialerOnce sync.Once

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
	dialerOnce.Do(func() {
		for {
			// 等待节点地址没设置则一直断网状态。
			target := remoteTarget(p.remoteURL)
			if target == "" {
				time.Sleep(time.Second)
				continue
			}
			v4cands, v6cands := physicalIPs()
			if b, ok := dialer.TCPDialer.(*dialer.BoundDialer); ok {
				if hasIP(v4cands, b.LocalIPv4) || hasIP(v6cands, b.LocalIPv6) {
					time.Sleep(5 * time.Second)
					continue
				}
			}
			v4, v6 := probe(v4cands, target), probe(v6cands, target)
			log.Println("Select Device v4", v4, "v6", v6)
			if v4 == nil && v6 == nil {
				time.Sleep(30 * time.Second)
				continue
			}
			d := &dialer.BoundDialer{LocalIPv4: v4, LocalIPv6: v6}
			dialer.TCPDialer, dialer.UDPDialer = d, d
			time.Sleep(time.Second)
		}
	})
}
