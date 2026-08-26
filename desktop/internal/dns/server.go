package dns

import (
	"fmt"
	"net"
	"sync"

	"github.com/miekg/dns"
)

var (
	serveMu   sync.Mutex
	serveConn *net.UDPConn
	serveDone chan struct{}
)

// StartUDP53 非阻塞启动 UDP DNS；已在运行则直接返回。
func StartUDP53(host string, port int) error {
	serveMu.Lock()
	defer serveMu.Unlock()
	if serveConn != nil {
		return nil
	}

	var ip net.IP
	if host != "" && host != "0.0.0.0" {
		ip = net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("invalid listen address: %s", host)
		}
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: ip, Port: port})
	if err != nil {
		return err
	}
	serveConn = conn
	serveDone = make(chan struct{})
	done := serveDone
	go func() {
		defer close(done)
		buf := make([]byte, 64*1024)
		for {
			n, clientAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			pkt := make([]byte, n)
			copy(pkt, buf[:n])
			go serveOne(conn, clientAddr, pkt)
		}
	}()
	return nil
}

// StopUDP53 停止 UDP DNS 监听。
func StopUDP53() {
	serveMu.Lock()
	conn := serveConn
	done := serveDone
	serveConn = nil
	serveDone = nil
	serveMu.Unlock()
	if conn == nil {
		return
	}
	_ = conn.Close()
	if done != nil {
		<-done
	}
}

func serveOne(conn *net.UDPConn, clientAddr *net.UDPAddr, pkt []byte) {
	req := new(dns.Msg)
	if err := req.Unpack(pkt); err != nil {
		return
	}
	resp, err := resolve(req)
	if err != nil {
		resp = new(dns.Msg)
		resp.SetRcode(req, dns.RcodeServerFailure)
	}
	data, err := resp.Pack()
	if err != nil {
		return
	}
	_, _ = conn.WriteToUDP(data, clientAddr)
}
