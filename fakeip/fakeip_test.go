package fakeip

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestAnswerFakeIPHasRRType(t *testing.T) {
	SetNeedFake(func(string) bool { return true })
	t.Cleanup(func() { SetNeedFake(nil) })

	for _, qtype := range []uint16{dns.TypeA, dns.TypeAAAA} {
		req := new(dns.Msg)
		req.SetQuestion("example.com.", qtype)
		query, err := req.Pack()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := Answer(query)
		if err != nil {
			t.Fatal(err)
		}
		resp := new(dns.Msg)
		if err := resp.Unpack(raw); err != nil {
			t.Fatal(err)
		}
		if len(resp.Answer) != 1 {
			t.Fatalf("qtype %d: answer count %d", qtype, len(resp.Answer))
		}
		rr := resp.Answer[0]
		if rr.Header().Rrtype != qtype {
			t.Fatalf("qtype %d: rrtype %d", qtype, rr.Header().Rrtype)
		}
		var ip net.IP
		switch v := rr.(type) {
		case *dns.A:
			ip = v.A
		case *dns.AAAA:
			ip = v.AAAA
		default:
			t.Fatalf("qtype %d: record %T", qtype, rr)
		}
		if ip == nil {
			t.Fatalf("qtype %d: nil address", qtype)
		}
		domain, ok := Lookup(ip)
		if !ok || domain != "example.com" {
			t.Fatalf("lookup %s: %q %v", ip, domain, ok)
		}
	}
}
