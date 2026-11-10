package dialer

import (
	"net"
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

var TCPDialer Dialer = &net.Dialer{}

var UDPDialer Dialer = &net.Dialer{}
