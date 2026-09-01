// Package fakeip is a UDP DNS server with Clash-style Fake-IP.
//
// Listen binds UDP DNS. SetNeedFake / SetProxy configure behavior after start
// (nil NeedFake means nothing is faked; nil SetProxy uses the default network).
// Domains that need fake are resolved via Google DoH. Real IPs and fake IPs are
// both stored in ttlmap with the DoH TTL; fake addresses increment and wrap.
package fakeip

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/penndev/gopkg/ttlmap"
	"github.com/penndev/prism/transport"
)

const udpTimeout = 5 * time.Second

type dnsEntry struct {
	IPs      []net.IP
	ExpireAt time.Time
}

type fakeEntry struct {
	Domain string
	IP     net.IP
}

// NeedFake reports whether domain should be answered with a fake IP.
type NeedFake func(domain string) bool

type Server struct {
	mu       sync.Mutex
	optMu    sync.RWMutex
	needFake NeedFake
	upstream string
	handle   transport.HandleConnect
	pool     *pool
	dnsMap   *ttlmap.Map
	fakeMap  *ttlmap.Map
	srv      *dns.Server
}

// Listen starts a UDP DNS server on address (host:port).
// upstream is a DNS IP or host:port; fakeNet is an IPv4 CIDR for fake addresses.
func Listen(address, upstream, fakeNet string) (*Server, error) {
	p, err := newPool(strings.TrimSpace(fakeNet))
	if err != nil {
		return nil, err
	}
	upstream = strings.TrimSpace(upstream)
	if upstream == "" {
		return nil, fmt.Errorf("upstream DNS required")
	}
	if _, _, err := net.SplitHostPort(upstream); err != nil {
		if net.ParseIP(upstream) == nil {
			return nil, fmt.Errorf("invalid upstream DNS: %s", upstream)
		}
		upstream = net.JoinHostPort(upstream, "53")
	}
	s := &Server{
		upstream: upstream,
		pool:     p,
		dnsMap:   ttlmap.New(),
		fakeMap:  ttlmap.New(),
	}

	started := make(chan struct{})
	errCh := make(chan error, 1)
	srv := &dns.Server{
		Addr:              address,
		Net:               "udp",
		Handler:           dns.HandlerFunc(s.serveDNS),
		NotifyStartedFunc: func() { close(started) },
	}
	s.srv = srv
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-started:
		return s, nil
	case err := <-errCh:
		if err == nil {
			err = fmt.Errorf("dns listen %s closed", address)
		}
		return nil, err
	}
}

func (s *Server) Close() error {
	if s == nil || s.srv == nil {
		return nil
	}
	return s.srv.Shutdown()
}

// SetNeedFake chooses which domains get a fake IP. nil means none.
func (s *Server) SetNeedFake(fn NeedFake) {
	if s == nil {
		return
	}
	s.optMu.Lock()
	defer s.optMu.Unlock()
	s.needFake = fn
}

// SetProxy routes Google DoH through handle. nil uses the default network.
func (s *Server) SetProxy(handle transport.HandleConnect) {
	if s == nil {
		return
	}
	s.optMu.Lock()
	defer s.optMu.Unlock()
	s.handle = handle
}

// Lookup returns the real domain and first real IP for a fake IP.
func (s *Server) Lookup(fakeIP string) (domain string, ip net.IP, ok bool) {
	ipAddr := net.ParseIP(strings.TrimSpace(fakeIP))
	if ipAddr == nil {
		return "", nil, false
	}
	if v4 := ipAddr.To4(); v4 != nil {
		ipAddr = v4
	}
	v, ok := s.fakeMap.Get("ip:" + ipAddr.String())
	if !ok {
		return "", nil, false
	}
	e, _ := v.(fakeEntry)
	domain = e.Domain
	if domain == "" {
		return "", nil, false
	}
	ips, _, err := s.lookupDNS(domain)
	if err != nil || len(ips) == 0 {
		return domain, nil, true
	}
	return domain, ips[0], true
}

func (s *Server) serveDNS(w dns.ResponseWriter, req *dns.Msg) {
	if req == nil || len(req.Question) == 0 {
		return
	}
	q := req.Question[0]
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(q.Name), "."))

	s.optMu.RLock()
	needFake := s.needFake
	s.optMu.RUnlock()
	if domain != "" && needFake != nil && needFake(domain) && (q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA) {
		resp, err := s.answerFake(req, domain, q.Qtype)
		if err != nil {
			resp = new(dns.Msg)
			resp.SetRcode(req, dns.RcodeServerFailure)
		}
		_ = w.WriteMsg(resp)
		return
	}

	resp, _, err := (&dns.Client{Net: "udp", Timeout: udpTimeout}).Exchange(req, s.upstream)
	if err != nil {
		resp = new(dns.Msg)
		resp.SetRcode(req, dns.RcodeServerFailure)
	}
	_ = w.WriteMsg(resp)
}

func (s *Server) answerFake(req *dns.Msg, domain string, qtype uint16) (*dns.Msg, error) {
	resp := new(dns.Msg)
	resp.SetReply(req)
	resp.Authoritative = true
	resp.RecursionAvailable = true
	if qtype == dns.TypeAAAA {
		return resp, nil
	}
	_, ttl, err := s.lookupDNS(domain)
	if err != nil {
		return nil, err
	}
	fake, err := s.assignFake(domain, ttl)
	if err != nil {
		return nil, err
	}
	resp.Answer = append(resp.Answer, &dns.A{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    uint32(ttl / time.Second),
		},
		A: fake,
	})
	return resp, nil
}

func (s *Server) assignFake(domain string, ttl time.Duration) (net.IP, error) {
	if v, ok := s.fakeMap.Get("h:" + domain); ok {
		e, _ := v.(fakeEntry)
		if e.IP != nil {
			return e.IP, nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.fakeMap.Get("h:" + domain); ok {
		e, _ := v.(fakeEntry)
		if e.IP != nil {
			return e.IP, nil
		}
	}
	n := s.pool.end - s.pool.start
	var ip net.IP
	for i := uint32(0); i < n; i++ {
		cand := s.pool.nextIP()
		if cand == nil {
			break
		}
		if _, taken := s.fakeMap.Get("ip:" + cand.String()); !taken {
			ip = cand
			break
		}
	}
	if ip == nil {
		ip = s.pool.nextIP()
		if ip == nil {
			return nil, fmt.Errorf("fake IP pool exhausted")
		}
		if old, ok := s.fakeMap.Get("ip:" + ip.String()); ok {
			e, _ := old.(fakeEntry)
			if e.Domain != "" {
				s.fakeMap.Delete("h:" + e.Domain)
			}
		}
	}
	if ttl > 0 {
		e := fakeEntry{Domain: domain, IP: ip}
		s.fakeMap.Set("h:"+domain, e, ttl)
		s.fakeMap.Set("ip:"+ip.String(), e, ttl)
	}
	return ip, nil
}

func (s *Server) lookupDNS(domain string) ([]net.IP, time.Duration, error) {
	if v, ok := s.dnsMap.Get(domain); ok {
		e, _ := v.(dnsEntry)
		if len(e.IPs) > 0 {
			ttl := time.Until(e.ExpireAt)
			if ttl < 0 {
				ttl = 0
			}
			return e.IPs, ttl, nil
		}
	}
	ips, ttl, err := s.queryGoogleDoH(domain)
	if err != nil {
		return nil, 0, err
	}
	if ttl > 0 {
		s.dnsMap.Set(domain, dnsEntry{IPs: ips, ExpireAt: time.Now().Add(ttl)}, ttl)
	}
	return ips, ttl, nil
}
