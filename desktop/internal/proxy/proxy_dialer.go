package proxy

import (
	"desktop/internal"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/penndev/prism/transport/dialer"
)

var dialerOnce sync.Once

func isTunInterfaceName(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "tun") || strings.Contains(lower, "wintun") || strings.Contains(lower, TUN_NAME)
}

func collectCandidateIPs() []net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	ips := make([]net.IP, 0)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isTunInterfaceName(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if v4 := ip.To4(); v4 != nil {
				ips = append(ips, v4)
			}
		}
	}
	return ips
}

func pickLocalIPForTarget(targetHost string) (net.IP, error) {
	candidates := collectCandidateIPs()
	if len(candidates) == 0 {
		return nil, errors.New("no available non-tun local ip")
	}
	for _, ip := range candidates {
		tcpDialer := net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: &net.TCPAddr{IP: ip},
		}
		conn, err := tcpDialer.Dial("tcp", targetHost)
		if err == nil {
			conn.Close()
			return ip, nil
		}
	}
	return nil, errors.New("no local ip can reach remote target")
}

func (p *Proxy) updateDialer() {
	localIP, err := pickLocalIPForTarget(net.JoinHostPort(p.remoteURL.Hostname(), p.remoteURL.Port()))
	if err != nil {
		internal.App.Event.Emit(
			internal.AppConfig.LogTypeName_LOG,
			"set dialer fallback: "+err.Error(),
		)
		return
	}

	dialer.TCPDialer = &net.Dialer{
		LocalAddr: &net.TCPAddr{IP: localIP},
	}
	dialer.UDPDialer = &net.Dialer{
		LocalAddr: &net.UDPAddr{IP: localIP},
	}
}
