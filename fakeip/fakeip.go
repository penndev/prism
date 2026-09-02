// Package fakeip is a UDP DNS server with Clash-style Fake-IP.
//
// Listen binds UDP DNS. SetNeedFake chooses which names get a fake A record
// (nil means none). Other queries are forwarded to the UDP upstream.
// Fake mappings last fakeTTL and wrap when the pool is exhausted.
package fakeip

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/penndev/gopkg/ttlmap"
)

const (
	udpTimeout = 5 * time.Second
	fakeTTL    = 10 * time.Second
)

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
	pool     *pool
	fakeMap  *ttlmap.Map
	srv      *dns.Server
}

// Match reports whether host is listed or is a subdomain of a listed domain.
func Match(host string, domains []string) bool {
	host = normName(host)
	if host == "" {
		return false
	}
	for _, d := range domains {
		d = normName(d)
		if d == "" {
			continue
		}
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func normName(s string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(s), "."))
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

func (s *Server) Addr() string {
	if s == nil || s.srv == nil {
		return ""
	}
	return s.srv.Addr
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

// SetUpstream sets the UDP DNS used for non-fake queries. Accepts IP or host:port.
func (s *Server) SetUpstream(up string) error {
	if s == nil {
		return fmt.Errorf("dns server is nil")
	}
	up = strings.TrimSpace(up)
	if up == "" {
		return fmt.Errorf("upstream DNS required")
	}
	if _, _, err := net.SplitHostPort(up); err != nil {
		if net.ParseIP(up) == nil {
			return fmt.Errorf("invalid upstream DNS: %s", up)
		}
		up = net.JoinHostPort(up, "53")
	}
	s.optMu.Lock()
	s.upstream = up
	s.optMu.Unlock()
	return nil
}

// Contains reports whether ip is in the fake-IP CIDR.
func (s *Server) Contains(ip string) bool {
	if s == nil || s.pool == nil {
		return false
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	return s.pool.contains(parsed)
}

// Lookup returns the domain for a fake IP.
func (s *Server) Lookup(fakeIP string) (domain string, ok bool) {
	if s == nil || s.fakeMap == nil {
		return "", false
	}
	ipAddr := net.ParseIP(strings.TrimSpace(fakeIP))
	if ipAddr == nil {
		return "", false
	}
	if v4 := ipAddr.To4(); v4 != nil {
		ipAddr = v4
	}
	v, ok := s.fakeMap.Get("ip:" + ipAddr.String())
	if !ok {
		return "", false
	}
	e, _ := v.(fakeEntry)
	if e.Domain == "" {
		return "", false
	}
	return e.Domain, true
}

func (s *Server) serveDNS(w dns.ResponseWriter, req *dns.Msg) {
	if req == nil || len(req.Question) == 0 {
		return
	}
	q := req.Question[0]
	domain := normName(q.Name)

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

	resp, _, err := (&dns.Client{Net: "udp", Timeout: udpTimeout}).Exchange(req, s.currentUpstream())
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
	fake, err := s.assignFake(domain)
	if err != nil {
		return nil, err
	}
	resp.Answer = append(resp.Answer, &dns.A{
		Hdr: dns.RR_Header{
			Name:   req.Question[0].Name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    uint32(fakeTTL / time.Second),
		},
		A: fake,
	})
	return resp, nil
}

func (s *Server) assignFake(domain string) (net.IP, error) {
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
	e := fakeEntry{Domain: domain, IP: ip}
	s.fakeMap.Set("h:"+domain, e, fakeTTL)
	s.fakeMap.Set("ip:"+ip.String(), e, fakeTTL)
	return ip, nil
}

func (s *Server) currentUpstream() string {
	s.optMu.RLock()
	defer s.optMu.RUnlock()
	return s.upstream
}
