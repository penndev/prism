package dns

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
)

// resolve：UDP 查询；若检测到旁路抢答则改用 DoH。
func resolve(req *dns.Msg) (*dns.Msg, error) {
	for _, q := range req.Question {
		log.Printf("DNS query: %s %s", q.Name, dns.TypeToString[q.Qtype])
	}

	udp, err := exchangeUDP(req)
	if err != nil {
		return nil, err
	}
	if udp.Msg == nil {
		return nil, fmt.Errorf("empty DNS response from %s", upstream)
	}

	resp := udp.Msg
	if udp.Conflict {
		log.Printf("DNS race hijack, re-querying via DoH")
		dohResp, err := queryDoH(req)
		if err != nil {
			return nil, err
		}
		resp = dohResp
	}

	logAnswers("DNS answer:", resp)
	return resp, nil
}
