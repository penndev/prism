package fakeip

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/penndev/gopkg/ttlmap"
	"github.com/penndev/prism/transport"
)

const (
	udpTimeout = 5 * time.Second
	fakeTTL    = 60
	mapTTL     = 30 * time.Minute
)

var (
	mu       sync.Mutex
	optMu    sync.RWMutex
	p        *pool
	hostToIP = ttlmap.New()
	ipToHost = ttlmap.New()

	needFake func(string) bool
	handle   transport.HandleConnect
	upstream = "8.8.8.8:53"
)

func init() {
	var err error
	p, err = newPool(defaultNet)
	if err != nil {
		panic(err)
	}
	handle = transport.Local()
}

func SetNeedFake(fn func(domain string) bool) {
	optMu.Lock()
	needFake = fn
	optMu.Unlock()
}

func SetHandleConnect(h transport.HandleConnect) {
	optMu.Lock()
	if h != nil {
		handle = h
	}
	optMu.Unlock()
}

func SetUpstream(dnsAddr string) {
	dnsAddr = strings.TrimSpace(dnsAddr)
	if dnsAddr == "" {
		return
	}
	if _, _, err := net.SplitHostPort(dnsAddr); err != nil {
		dnsAddr = net.JoinHostPort(dnsAddr, "53")
	}
	optMu.Lock()
	upstream = dnsAddr
	optMu.Unlock()
}

func Lookup(fakeIP string) (string, bool) {
	ip := net.ParseIP(strings.TrimSpace(fakeIP))
	if ip == nil {
		return "", false
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	d, ok := ipToHost.Get(ip.String())
	if !ok {
		return "", false
	}
	s, _ := d.(string)
	return s, s != ""
}

func Handle(conn net.Conn, network, address string) error {
	defer conn.Close()
	buf := make([]byte, 4096)
	_ = conn.SetDeadline(time.Now().Add(udpTimeout))
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}
		out, err := answer(buf[:n])
		if err != nil || len(out) == 0 {
			continue
		}
		if _, err := conn.Write(out); err != nil {
			return err
		}
		_ = conn.SetDeadline(time.Now().Add(udpTimeout))
	}
}

func answer(query []byte) ([]byte, error) {
	req := new(dns.Msg)
	if err := req.Unpack(query); err != nil {
		return nil, err
	}
	if len(req.Question) == 0 {
		return nil, nil
	}
	q := req.Question[0]
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(q.Name), "."))

	optMu.RLock()
	fn := needFake
	optMu.RUnlock()
	if name != "" && fn != nil && fn(name) && (q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA) {
		resp := new(dns.Msg)
		resp.SetReply(req)
		resp.Authoritative = true
		resp.RecursionAvailable = true
		if q.Qtype == dns.TypeA {
			ip := assign(name)
			if ip != nil {
				resp.Answer = append(resp.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: fakeTTL},
					A:   ip,
				})
			}
		}
		log.Println("dns fake:", resp)
		return resp.Pack()
	}
	return forward(req)
}

func assign(domain string) net.IP {
	mu.Lock()
	defer mu.Unlock()
	if v, ok := hostToIP.Get(domain); ok {
		if ip, ok := v.(net.IP); ok && ip != nil {
			hostToIP.Set(domain, ip, mapTTL)
			ipToHost.Set(ip.String(), domain, mapTTL)
			return ip
		}
	}
	ip := p.nextIP()
	if ip == nil {
		return nil
	}
	if old, ok := ipToHost.Get(ip.String()); ok {
		if s, ok := old.(string); ok {
			hostToIP.Delete(s)
		}
	}
	hostToIP.Set(domain, ip, mapTTL)
	ipToHost.Set(ip.String(), domain, mapTTL)
	return ip
}

func forward(req *dns.Msg) ([]byte, error) {
	optMu.RLock()
	h := handle
	up := upstream
	optMu.RUnlock()
	if h == nil {
		h = transport.Local()
	}
	packed, err := req.Pack()
	if err != nil {
		return nil, err
	}
	left, right := net.Pipe()
	go func() {
		_ = h(right, "udp", up)
		_ = right.Close()
	}()
	defer left.Close()
	_ = left.SetDeadline(time.Now().Add(udpTimeout))
	if _, err := left.Write(packed); err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := left.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
