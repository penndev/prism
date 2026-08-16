package dialer

import (
	"net"
	"strings"
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

var TCPDialer Dialer = &net.Dialer{}

var UDPDialer Dialer = &net.Dialer{}

// BoundDialer 按目标地址族选择本地 IPv4 / IPv6，避免交叉绑定导致 no suitable address found。
type BoundDialer struct {
	LocalIPv4 net.IP
	LocalIPv6 net.IP
	Zone      string
}

func (d *BoundDialer) Dial(network, address string) (net.Conn, error) {
	nd := net.Dialer{}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nd.Dial(network, address)
	}
	name, destZone, _ := strings.Cut(host, "%")
	ip := net.ParseIP(name)

	var localIP net.IP
	var localZone string
	switch {
	case ip == nil && d.LocalIPv4 != nil:
		localIP = d.LocalIPv4
	case ip == nil:
		localIP = d.LocalIPv6
		localZone = d.Zone
	case ip.To4() != nil:
		localIP = d.LocalIPv4
	default:
		localIP = d.LocalIPv6
		localZone = d.Zone
		if destZone == "" && d.Zone != "" && ip.IsLinkLocalUnicast() {
			address = net.JoinHostPort(ip.String()+"%"+d.Zone, port)
		}
	}
	if localIP != nil {
		if strings.HasPrefix(network, "udp") {
			nd.LocalAddr = &net.UDPAddr{IP: localIP, Zone: localZone}
		} else {
			nd.LocalAddr = &net.TCPAddr{IP: localIP, Zone: localZone}
		}
	}
	return nd.Dial(network, address)
}
