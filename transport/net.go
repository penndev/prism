package transport

import (
	"net"

	"github.com/penndev/gopkg/util"
	"github.com/penndev/prism/transport/dialer"
)

var localHandle HandleConnect

func init() {
	localHandle = Local()
}

// 本地请求，不用远程
func Local() HandleConnect {
	return func(conn net.Conn, network, address string) error {
		var d dialer.Dialer
		switch network {
		case "tcp":
			d = dialer.TCPDialer
		case "udp":
			d = dialer.UDPDialer
		}
		if d == nil {
			return net.UnknownNetworkError(network)
		}
		remote, err := d.Dial(network, address)
		if err != nil {
			return err
		}
		util.Pipe(conn, remote)
		return nil
	}
}
