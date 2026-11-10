package network

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func dialOutbound(network, address string) (net.Conn, error) {
	return boundDial.Dial(network, address)
}

func bindSocket(ipv6 bool, index int, raw syscall.RawConn) error {
	level, opt := syscall.IPPROTO_IP, unix.IP_BOUND_IF
	if ipv6 {
		level, opt = syscall.IPPROTO_IPV6, unix.IPV6_BOUND_IF
	}
	var sockErr error
	err := raw.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInt(int(fd), level, opt, index)
	})
	if err != nil {
		return err
	}
	return sockErr
}
