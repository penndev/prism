package engine

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/penndev/prism/fakeip"
)

type dnsLogHook struct {
	name   string
	qtype  string
	result string
}

func (h *dnsLogHook) OnStatus(string)                     {}
func (h *dnsLogHook) OnConnection(string, string, string) {}
func (h *dnsLogHook) OnDNS(name, qtype, result string) {
	h.name = name
	h.qtype = qtype
	h.result = result
}
func (h *dnsLogHook) UseProxy(string, string) bool { return false }
func (h *dnsLogHook) OnProxyRead(int64)            {}
func (h *dnsLogHook) OnProxyWrite(int64)           {}

func TestRelayDNSLogsSynthesizedAnswer(t *testing.T) {
	fakeip.SetNeedFake(func(string) bool { return true })
	t.Cleanup(func() { fakeip.SetNeedFake(nil) })

	client, server := net.Pipe()
	hook := &dnsLogHook{}
	finished := make(chan struct{})
	go func() {
		relayDNS(hook, server)
		close(finished)
	}()
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	request := new(dns.Msg)
	request.SetQuestion("example.com.", dns.TypeAAAA)
	query, err := request.Pack()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Write(query); err != nil {
		t.Fatal(err)
	}

	buffer := make([]byte, 4096)
	n, err := client.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}
	response := new(dns.Msg)
	if err := response.Unpack(buffer[:n]); err != nil {
		t.Fatal(err)
	}
	if len(response.Answer) != 1 {
		t.Fatalf("answer count %d", len(response.Answer))
	}
	answer, ok := response.Answer[0].(*dns.AAAA)
	if !ok {
		t.Fatalf("answer type %T", response.Answer[0])
	}
	if hook.name != "example.com" || hook.qtype != "AAAA" || hook.result != answer.AAAA.String() {
		t.Fatalf("DNS log = (%q, %q, %q)", hook.name, hook.qtype, hook.result)
	}

	_ = client.Close()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("DNS relay did not stop after the client closed")
	}
}
