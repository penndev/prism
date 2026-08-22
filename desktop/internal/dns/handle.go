package dns

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
)

var upstream = "8.8.8.8"

func handle(req dns.Msg) (dns.Msg, error) {
	for _, q := range req.Question {
		log.Printf("DNS query: %s %s", q.Name, dns.TypeToString[q.Qtype])
	}

	client := &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	resp, _, err := client.Exchange(&req, net.JoinHostPort(upstream, "53"))
	if err != nil {
		return dns.Msg{}, err
	}
	if resp == nil {
		return dns.Msg{}, fmt.Errorf("empty DNS response from %s", upstream)
	}

	if rcode := dns.RcodeToString[resp.Rcode]; rcode != "NOERROR" {
		log.Printf("DNS rcode: %s", rcode)
	}
	for _, rr := range resp.Answer {
		log.Printf("DNS answer: %s", rr.String())
	}
	return *resp, nil
}
