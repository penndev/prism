package network

import (
	"context"
	"desktop/internal/route"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"syscall"
)

// Dialer 在 connect 之前把 socket 钉到物理网卡。
type Dialer struct{}

func (Dialer) Dial(network, address string) (net.Conn, error) {
	return dialOutbound(network, address)
}

var boundDial = net.Dialer{
	Control:  controlBind,
	Resolver: &net.Resolver{PreferGo: true, Dial: dialDNS},
}

// dialDNS 忽略 Go 传进来的服务器：PreferGo 会把 TUN 上的 DNS 也列进来，
// 响应被截断后的 TCP 重试同样走这里。
func dialDNS(ctx context.Context, network, _ string) (net.Conn, error) {
	server := route.NextDNS()
	return dialBound(ctx, network, net.JoinHostPort(server, "53"), net.ParseIP(server))
}

// dialBound 对 TCP 额外设置 LocalAddr。
// Windows 上 IP_UNICAST_IF / IPV6_UNICAST_IF 只保证 UDP，ConnectEx 前 Go 还会 bind 到通配地址。
func dialBound(ctx context.Context, network, address string, ip net.IP) (net.Conn, error) {
	d := net.Dialer{Control: controlBind}
	if strings.HasPrefix(network, "tcp") && ip != nil && !leaveUnbound(ip) {
		if addr, ok := netip.AddrFromSlice(ip); ok {
			if src := route.SourceIP(addr.Unmap()); src != nil {
				d.LocalAddr = &net.TCPAddr{IP: src}
			}
		}
	}
	return d.DialContext(ctx, network, address)
}

func controlBind(network, address string, c syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	host, _, _ = strings.Cut(host, "%")
	ip := net.ParseIP(host)
	if ip != nil && leaveUnbound(ip) {
		return nil
	}

	// 网络名带 4/6 后缀时以它为准，否则看目标地址。
	ipv6 := strings.HasSuffix(network, "6") ||
		(!strings.HasSuffix(network, "4") && ip != nil && ip.To4() == nil)

	iface := route.Load().Iface(ipv6)
	if iface == nil {
		return errUnreachable(network, address)
	}
	return bindSocket(ipv6, iface.Index, c)
}

// leaveUnbound 判断目标是否不该绑到物理网卡（IsMulticast 已包含链路本地/接口本地组播）。
func leaveUnbound(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast()
}

func errUnreachable(network, address string) error {
	return fmt.Errorf("%s %s: network unreachable", network, address)
}
