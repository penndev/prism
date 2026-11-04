package fakeip

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/penndev/gopkg/ttlmap"
	"github.com/penndev/prism/transport"
)

const (
	upstreamTimeout = 5 * time.Second
	dnsAnswerTTL    = 60
	addressMapTTL   = 30 * time.Minute
	// FakeIPv4CIDR is the IPv4 range used for synthetic addresses.
	FakeIPv4CIDR = "198.18.0.0/15"
	// FakeIPv6CIDR is the IPv6 range used for synthetic addresses.
	FakeIPv6CIDR = "fd19:ffff:198:18::/64"

	// DefaultUpstream is used when no DNS server has been configured.
	DefaultUpstream = "8.8.8.8:53"
)

var (
	configMu      sync.RWMutex
	ipv4Allocator addressAllocator
	ipv6Allocator addressAllocator

	shouldFakeDomain func(string) bool
	dnsForwarder     transport.HandleConnect
	dnsUpstream      = DefaultUpstream
)

func init() {
	var err error
	ipv4Pool, err := newIPv4Pool(FakeIPv4CIDR)
	if err != nil {
		panic(err)
	}
	ipv4Allocator = addressAllocator{
		pool:          ipv4Pool,
		hostToAddress: ttlmap.New(),
		addressToHost: ttlmap.New(),
	}
	ipv6Pool, err := newIPv6Pool(FakeIPv6CIDR)
	if err != nil {
		panic(err)
	}
	ipv6Allocator = addressAllocator{
		pool:          ipv6Pool,
		hostToAddress: ttlmap.New(),
		addressToHost: ttlmap.New(),
	}
	dnsForwarder = transport.Local()
}

// SetNeedFake sets the predicate that selects domains for synthetic answers.
func SetNeedFake(shouldFake func(domain string) bool) {
	configMu.Lock()
	shouldFakeDomain = shouldFake
	configMu.Unlock()
}

// SetHandleConnect sets the connection handler used to reach the DNS upstream.
func SetHandleConnect(connect transport.HandleConnect) {
	configMu.Lock()
	if connect != nil {
		dnsForwarder = connect
	}
	configMu.Unlock()
}

// SetUpstream sets the DNS server for queries that are not answered locally.
func SetUpstream(address string) {
	address = strings.TrimSpace(address)
	if address == "" {
		return
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, "53")
	}
	configMu.Lock()
	dnsUpstream = address
	configMu.Unlock()
}

// Lookup returns the domain associated with a synthetic IPv4 or IPv6 address.
func Lookup(ip net.IP) (string, bool) {
	if ip == nil {
		return "", false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		value, ok := ipv4Allocator.addressToHost.Get(ipv4.String())
		if !ok {
			return "", false
		}
		domain, _ := value.(string)
		return domain, domain != ""
	}
	value, ok := ipv6Allocator.addressToHost.Get(ip.String())
	if !ok {
		return "", false
	}
	domain, _ := value.(string)
	return domain, domain != ""
}

// Answer handles a DNS wire-format query, synthesizing addresses for selected domains.
func Answer(query []byte) ([]byte, error) {
	request := new(dns.Msg)
	if err := request.Unpack(query); err != nil {
		return nil, err
	}
	if len(request.Question) == 0 {
		return nil, nil
	}

	question := request.Question[0]
	domain := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(question.Name), "."))
	configMu.RLock()
	shouldFake := shouldFakeDomain
	configMu.RUnlock()
	if domain != "" && shouldFake != nil && shouldFake(domain) {
		var record dns.RR
		header := dns.RR_Header{
			Name:   question.Name,
			Rrtype: question.Qtype,
			Class:  dns.ClassINET,
			Ttl:    dnsAnswerTTL,
		}
		switch question.Qtype {
		case dns.TypeA:
			if address := ipv4Allocator.assign(domain); address != nil {
				record = &dns.A{Hdr: header, A: address}
			}
		case dns.TypeAAAA:
			if address := ipv6Allocator.assign(domain); address != nil {
				record = &dns.AAAA{Hdr: header, AAAA: address}
			}
		}
		if question.Qtype == dns.TypeA || question.Qtype == dns.TypeAAAA {
			response := new(dns.Msg)
			response.SetReply(request)
			response.Authoritative = true
			response.RecursionAvailable = true
			if record != nil {
				response.Answer = append(response.Answer, record)
			}
			return response.Pack()
		}
	}

	configMu.RLock()
	connect := dnsForwarder
	upstream := dnsUpstream
	configMu.RUnlock()
	if connect == nil {
		connect = transport.Local()
	}

	packed, err := request.Pack()
	if err != nil {
		return nil, err
	}
	client, server := net.Pipe()
	go func() {
		_ = connect(server, "udp", upstream)
		_ = server.Close()
	}()
	defer client.Close()
	_ = client.SetDeadline(time.Now().Add(upstreamTimeout))
	if _, err := client.Write(packed); err != nil {
		return nil, err
	}
	response := make([]byte, 4096)
	n, err := client.Read(response)
	if err != nil {
		return nil, err
	}
	return response[:n], nil
}
