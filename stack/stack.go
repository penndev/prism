package stack

import (
	"log"
	"net"
	"runtime"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

type ForwarderUDPRequest struct {
	Conn       net.Conn
	RemoteAddr net.UDPAddr
	LocalAddr  net.UDPAddr
}

type ForwarderTCPRequest struct {
	Conn       net.Conn
	RemoteAddr net.TCPAddr
	LocalAddr  net.TCPAddr
}

type Option struct {
	HandleTCP  func(*ForwarderTCPRequest)
	HandlerUDP func(*ForwarderUDPRequest)
	EndPoint   stack.LinkEndpoint
}

func New(option Option) {
	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
			icmp.NewProtocol6,
		},
	})

	nicID := tcpip.NICID(s.NextNICID())
	s.CreateNICWithOptions(nicID, option.EndPoint, stack.NICOptions{})
	s.SetPromiscuousMode(nicID, true)
	s.SetSpoofing(nicID, true)
	s.SetRouteTable([]tcpip.Route{
		{Destination: header.IPv4EmptySubnet, NIC: nicID},
		{Destination: header.IPv6EmptySubnet, NIC: nicID},
	})

	// Windows 上 RACK 会让 gVisor TCP 严重掉速，sing-tun / sing-box 同样关掉。
	if runtime.GOOS == "windows" {
		if err := s.SetTransportProtocolOption(tcp.ProtocolNumber, new(tcpip.TCPRecovery)); err != nil {
			log.Fatal("TCPRecovery: ", err)
		}
	}

	// handle TCP setting
	if option.HandleTCP != nil {
		tcpForwarder := tcp.NewForwarder(s, 0, 2048, func(r *tcp.ForwarderRequest) {
			var ftr ForwarderTCPRequest
			var waiterQueue waiter.Queue
			if endPoint, err := r.CreateEndpoint(&waiterQueue); err == nil {
				ftr.Conn = gonet.NewTCPConn(&waiterQueue, endPoint)
			} else {
				r.Complete(true)
				return
			}
			defer r.Complete(false)
			addrInfo := r.ID()
			ftr.LocalAddr = net.TCPAddr{
				IP:   addrInfo.RemoteAddress.AsSlice(),
				Port: int(addrInfo.RemotePort),
			}
			ftr.RemoteAddr = net.TCPAddr{
				IP:   addrInfo.LocalAddress.AsSlice(),
				Port: int(addrInfo.LocalPort),
			}

			go option.HandleTCP(&ftr)
		})
		s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)
	}

	if option.HandlerUDP != nil {
		udpForwarder := udp.NewForwarder(s, func(r *udp.ForwarderRequest) {
			var fur ForwarderUDPRequest
			var waiterQueue waiter.Queue
			if endPoint, err := r.CreateEndpoint(&waiterQueue); err == nil {
				fur.Conn = gonet.NewUDPConn(&waiterQueue, endPoint)
			} else {
				return
			}
			addrInfo := r.ID()
			fur.LocalAddr = net.UDPAddr{
				IP:   addrInfo.RemoteAddress.AsSlice(),
				Port: int(addrInfo.RemotePort),
			}
			fur.RemoteAddr = net.UDPAddr{
				IP:   addrInfo.LocalAddress.AsSlice(),
				Port: int(addrInfo.LocalPort),
			}
			go option.HandlerUDP(&fur)
		})
		s.SetTransportProtocolHandler(udp.ProtocolNumber, udpForwarder.HandlePacket)
	}
}
