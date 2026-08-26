package dns

import (
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

// ListenUDP53 在 UDP 53 上监听 DNS 查询。host 为 0.0.0.0 或空时监听全部网卡。
func ListenUDP53(host string) error {
	var ip net.IP
	if host != "" && host != "0.0.0.0" {
		ip = net.ParseIP(host)
		if ip == nil {
			return fmt.Errorf("invalid listen address: %s", host)
		}
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: ip, Port: 53})
	if err != nil {
		return err
	}
	defer conn.Close()

	log.Printf("DNS server listening on UDP %s", conn.LocalAddr())

	buf := make([]byte, 64*1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("DNS read error:", err)
			continue
		}
		pkt := make([]byte, n)
		copy(pkt, buf[:n])
		go serveOne(conn, clientAddr, pkt)
	}
}

func serveOne(conn *net.UDPConn, clientAddr *net.UDPAddr, pkt []byte) {
	req := new(dns.Msg)
	if err := req.Unpack(pkt); err != nil {
		log.Println("DNS unpack error:", err)
		return
	}

	resp, err := resolve(req)
	if err != nil {
		log.Println("DNS resolve error:", err)
		fail := new(dns.Msg)
		fail.SetRcode(req, dns.RcodeServerFailure)
		resp = fail
	}

	data, err := resp.Pack()
	if err != nil {
		log.Println("DNS pack error:", err)
		return
	}
	if _, err = conn.WriteToUDP(data, clientAddr); err != nil {
		log.Println("DNS write error:", err)
	}
}
