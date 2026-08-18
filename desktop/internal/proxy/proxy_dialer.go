package proxy

import (
	"desktop/internal"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/penndev/prism/transport/dialer"
)

func addrIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}

func isTunIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	if TUN_IP.IsValid() && TUN_IP.Contains(addr) {
		return true
	}
	return TUN_IP6.IsValid() && TUN_IP6.Contains(addr)
}

func collectPhysicalIPs() (v4, v6 []net.IP) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}
	tunName := strings.ToLower(TUN_NAME)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		if tunName != "" && (name == tunName || strings.HasPrefix(name, tunName) || strings.Contains(name, "wintun")) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := addrIP(addr)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || isTunIP(ip) {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				v4 = append(v4, ip4)
			} else {
				v6 = append(v6, ip)
			}
		}
	}
	return v4, v6
}

func dialFrom(ip net.IP, target string) bool {
	c, err := (&net.Dialer{
		Timeout:   2 * time.Second,
		LocalAddr: &net.TCPAddr{IP: ip},
	}).Dial("tcp", target)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func probe(cands []net.IP, target string) net.IP {
	host, _, _ := net.SplitHostPort(target)
	dest := net.ParseIP(host)
	for _, ip := range cands {
		if dest != nil && (dest.To4() == nil) != (ip.To4() == nil) {
			continue
		}
		if dialFrom(ip, target) {
			return ip
		}
	}
	if len(cands) > 0 {
		return cands[0]
	}
	return nil
}

var dialerOnce sync.Once

func (p *Proxy) updateDialer() {
	dialerOnce.Do(func() {
		for {
			// 未设置代理不用处理网卡
			if p.remoteURL == nil || p.remoteURL.Hostname() == "" || p.remoteURL.Port() == "" {
				time.Sleep(time.Second)
				continue
			}
			target := net.JoinHostPort(p.remoteURL.Hostname(), p.remoteURL.Port())

			// 当前绑的物理 IP 还能连上节点就不动。
			if b, ok := dialer.TCPDialer.(*dialer.BoundDialer); ok {
				ip := b.LocalIPv4
				if ip == nil {
					ip = b.LocalIPv6
				}
				if ip != nil && dialFrom(ip, target) {
					time.Sleep(30 * time.Second)
					continue
				}
			}
			// 获取新的物理 网卡连接
			v4cands, v6cands := collectPhysicalIPs()
			v4, v6 := probe(v4cands, target), probe(v6cands, target)
			if v4 == nil && v6 == nil {
				internal.App.Event.Emit(internal.AppConfig.LogTypeName_LOG, "set dialer fallback: no available local ip")
				time.Sleep(30 * time.Second)
				continue
			}

			log.Println("v4", v4, "v6", v6)
			bound := &dialer.BoundDialer{LocalIPv4: v4, LocalIPv6: v6}
			dialer.TCPDialer = bound
			dialer.UDPDialer = bound
			time.Sleep(30 * time.Second)
		}
	})
}
