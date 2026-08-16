package proxy

import (
	"desktop/internal"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/penndev/prism/transport/dialer"
)

var dialerOnce sync.Once

type localIP struct {
	IP   net.IP
	Zone string
}

func collectCandidateIPs() (v4, v6 []localIP) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		zone := strconv.Itoa(iface.Index)
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				v4 = append(v4, localIP{IP: ip4})
				continue
			}
			if ip.To16() == nil {
				continue
			}
			li := localIP{IP: ip}
			if ip.IsLinkLocalUnicast() {
				li.Zone = zone
			}
			v6 = append(v6, li)
		}
	}
	return v4, v6
}

func probeLocalIP(cands []localIP, targetHost string) (net.IP, string) {
	host, port, err := net.SplitHostPort(targetHost)
	if err != nil {
		return nil, ""
	}
	name, targetZone, found := strings.Cut(host, "%")
	if !found {
		name = host
	}
	targetIP := net.ParseIP(name)
	for _, c := range cands {
		if targetIP != nil && (targetIP.To4() == nil) != (c.IP.To4() == nil) {
			continue
		}
		dest := targetHost
		if targetIP != nil && targetIP.To4() == nil && targetIP.IsLinkLocalUnicast() && targetZone == "" && c.Zone != "" {
			dest = net.JoinHostPort(targetIP.String()+"%"+c.Zone, port)
		}
		tcpDialer := net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 30 * time.Second,
			LocalAddr: &net.TCPAddr{IP: c.IP, Zone: c.Zone},
		}
		conn, err := tcpDialer.Dial("tcp", dest)
		if err != nil {
			continue
		}
		conn.Close()
		return c.IP, c.Zone
	}
	return nil, ""
}

func pickLocalIPForTarget(targetHost string) (v4, v6 net.IP, zone string, err error) {
	v4cands, v6cands := collectCandidateIPs()
	if len(v4cands) == 0 && len(v6cands) == 0 {
		return nil, nil, "", errors.New("no available local ip")
	}
	v4, _ = probeLocalIP(v4cands, targetHost)
	if v4 == nil && len(v4cands) > 0 {
		v4 = v4cands[0].IP
	}
	v6, zone = probeLocalIP(v6cands, targetHost)
	if v6 == nil && len(v6cands) > 0 {
		v6 = v6cands[0].IP
		zone = v6cands[0].Zone
	}
	return v4, v6, zone, nil
}

func (p *Proxy) updateDialer() {
	dialerOnce.Do(func() {

		// 循环设置检查心跳。设置出网网卡的IP。来应对网络变化。比如无线切有线
		// 检查应对的目标服务器是 p.remoteURL
		for {
			if p.remoteURL == nil { // 等待设置远程代理信息。
				time.Sleep(1 * time.Second)
				continue
			}

			host := p.remoteURL.Hostname()
			port := p.remoteURL.Port()
			if host == "" || port == "" {
				time.Sleep(1 * time.Second)
				continue
			}

			v4, v6, zone, err := pickLocalIPForTarget(net.JoinHostPort(host, port))
			if err != nil {
				internal.App.Event.Emit(
					internal.AppConfig.LogTypeName_LOG,
					"set dialer fallback: "+err.Error(),
				)
				time.Sleep(30 * time.Second)
				continue
			}

			bound := &dialer.BoundDialer{
				LocalIPv4: v4,
				LocalIPv6: v6,
				Zone:      zone,
			}
			dialer.TCPDialer = bound
			dialer.UDPDialer = bound

			time.Sleep(30 * time.Second)
		}

	})
}
