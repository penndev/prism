package dns

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type udpResult struct {
	Msg      *dns.Msg
	Conflict bool // 短窗口内收到与首包 Answer 不一致的旁路抢答
}

func answerKey(msg *dns.Msg) string {
	var b strings.Builder
	for _, rr := range msg.Answer {
		b.WriteString(rr.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func logAnswers(prefix string, msg *dns.Msg) {
	log.Printf("%s", prefix)
	for _, rr := range msg.Answer {
		log.Printf("  %s", rr.String())
	}
}

// exchangeUDP 向上游查询；首包后再等 raceWindow，检测旁路抢答。
func exchangeUDP(req *dns.Msg) (udpResult, error) {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(upstream, "53"), udpTimeout)
	if err != nil {
		return udpResult{}, err
	}
	defer conn.Close()

	data, err := req.Pack()
	if err != nil {
		return udpResult{}, err
	}
	_ = conn.SetDeadline(time.Now().Add(udpTimeout))
	if _, err := conn.Write(data); err != nil {
		return udpResult{}, err
	}

	buf := make([]byte, 64*1024)
	var first *dns.Msg
	raceDeadline := time.Time{}

	for {
		if !raceDeadline.IsZero() {
			_ = conn.SetReadDeadline(raceDeadline)
		}
		n, err := conn.Read(buf)
		if err != nil {
			if first != nil {
				return udpResult{Msg: first}, nil
			}
			return udpResult{}, err
		}

		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil || msg.Id != req.Id {
			continue
		}

		from := conn.RemoteAddr().String()
		if first == nil {
			first = msg
			logAnswers(fmt.Sprintf("DNS reply#1 from %s:", from), msg)
			raceDeadline = time.Now().Add(raceWindow)
			continue
		}

		logAnswers(fmt.Sprintf("DNS reply#2 (race) from %s:", from), msg)
		conflict := answerKey(msg) != answerKey(first)
		if conflict {
			log.Printf("DNS race conflict")
		} else {
			log.Printf("DNS race: duplicate")
		}
		return udpResult{Msg: first, Conflict: conflict}, nil
	}
}
