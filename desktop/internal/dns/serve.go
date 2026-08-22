package dns

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/miekg/dns"
)

// Serve 在 UDP 53 上监听 DNS 查询。host 为 0.0.0.0 或空时监听全部网卡。
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
			log.Println("read error:", err)
			continue
		}

		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			log.Println("DNS unpack error:", err)
			continue
		}

		resp, err := handle(*msg)
		if err != nil {
			log.Println("DNS handle error:", err)
			fail := new(dns.Msg)
			fail.SetRcode(msg, dns.RcodeServerFailure)
			resp = *fail
		}

		data, err := resp.Pack()
		if err != nil {
			log.Println("DNS pack error:", err)
			continue
		}

		if _, err = conn.WriteToUDP(data, clientAddr); err != nil {
			log.Println("write error:", err)
		}
	}
}

func HandleDOH(w http.ResponseWriter, r *http.Request) {

}

func Handleresolve(w http.ResponseWriter, r *http.Request) {

}
