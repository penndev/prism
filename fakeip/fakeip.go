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
	fakeIPv4CIDR    = "198.18.0.0/15"
	fakeIPv6CIDR    = "fd19:ffff:198:18::/64"

	// DefaultUpstream is used when no DNS server has been configured.
	DefaultUpstream = "8.8.8.8:53"
)

var (
	addressMu sync.Mutex
	configMu  sync.RWMutex
	ipv4Pool  *addressPool
	ipv6Pool  *addressPool

	hostToIPv4 = ttlmap.New()
	ipv4ToHost = ttlmap.New()
	hostToIPv6 = ttlmap.New()
	ipv6ToHost = ttlmap.New()

	shouldFakeDomain func(string) bool
	dnsForwarder     transport.HandleConnect
	dnsUpstream      = DefaultUpstream
)

func init() {
	var err error
	ipv4Pool, err = newIPv4Pool(fakeIPv4CIDR)
	if err != nil {
		panic(err)
	}
	ipv6Pool, err = newIPv6Pool(fakeIPv6CIDR)
	if err != nil {
		panic(err)
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
func Lookup(address string) (string, bool) {
	ip := net.ParseIP(strings.TrimSpace(address))
	if ip == nil {
		return "", false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		value, ok := ipv4ToHost.Get(ipv4.String())
		if !ok {
			return "", false
		}
		domain, _ := value.(string)
		return domain, domain != ""
	}
	value, ok := ipv6ToHost.Get(ip.String())
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
		if response, handled, err := synthesizeAnswer(request, question, domain); handled || err != nil {
			return response, err
		}
	}
	return forward(request)
}

func synthesizeAnswer(request *dns.Msg, question dns.Question, domain string) ([]byte, bool, error) {
	var record dns.RR
	switch question.Qtype {
	case dns.TypeA:
		if address := assignAddress(domain, ipv4Pool, hostToIPv4, ipv4ToHost); address != nil {
			record = &dns.A{A: address}
		}
	case dns.TypeAAAA:
		if address := assignAddress(domain, ipv6Pool, hostToIPv6, ipv6ToHost); address != nil {
			record = &dns.AAAA{AAAA: address}
		}
	default:
		return nil, false, nil
	}

	response := new(dns.Msg)
	response.SetReply(request)
	response.Authoritative = true
	response.RecursionAvailable = true
	if record != nil {
		record.Header().Name = question.Name
		record.Header().Class = dns.ClassINET
		record.Header().Ttl = dnsAnswerTTL
		response.Answer = append(response.Answer, record)
	}
	packed, err := response.Pack()
	return packed, true, err
}

func assignAddress(domain string, pool *addressPool, hostToAddress, addressToHost interface {
	Get(string) (any, bool)
	Set(string, any, time.Duration)
	Delete(any)
}) net.IP {
	addressMu.Lock()
	defer addressMu.Unlock()

	if value, ok := hostToAddress.Get(domain); ok {
		if address, ok := value.(net.IP); ok && address != nil {
			hostToAddress.Set(domain, address, addressMapTTL)
			addressToHost.Set(address.String(), domain, addressMapTTL)
			return address
		}
	}

	address := pool.nextIP()
	if address == nil {
		return nil
	}
	if value, ok := addressToHost.Get(address.String()); ok {
		if previousDomain, ok := value.(string); ok {
			hostToAddress.Delete(previousDomain)
		}
	}
	hostToAddress.Set(domain, address, addressMapTTL)
	addressToHost.Set(address.String(), domain, addressMapTTL)
	return address
}

func forward(request *dns.Msg) ([]byte, error) {
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
