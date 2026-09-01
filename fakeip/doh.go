package fakeip

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

const (
	googleDoH  = "https://8.8.8.8/dns-query"
	dohTimeout = 5 * time.Second
)

func (s *Server) queryGoogleDoH(domain string) ([]net.IP, time.Duration, error) {
	req := new(dns.Msg)
	req.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	req.RecursionDesired = true
	payload, err := req.Pack()
	if err != nil {
		return nil, 0, err
	}

	httpReq, err := http.NewRequest(http.MethodPost, googleDoH, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")

	tr := &http.Transport{}
	s.optMu.RLock()
	handle := s.handle
	s.optMu.RUnlock()
	if handle != nil {
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			local, remote := net.Pipe()
			go func() {
				err := handle(remote, network, addr)
				_ = remote.Close()
				if err != nil {
					_ = local.Close()
				}
			}()
			if deadline, ok := ctx.Deadline(); ok {
				_ = local.SetDeadline(deadline)
			}
			return local, nil
		}
	}

	httpResp, err := (&http.Client{Timeout: dohTimeout, Transport: tr}).Do(httpReq)
	if err != nil {
		return nil, 0, err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("DoH: %s", httpResp.Status)
	}
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, 0, err
	}

	resp := new(dns.Msg)
	if err := resp.Unpack(body); err != nil {
		return nil, 0, err
	}
	if resp.Rcode != dns.RcodeSuccess {
		return nil, 0, fmt.Errorf("DoH rcode %d", resp.Rcode)
	}

	var ips []net.IP
	ttl := time.Duration(0)
	for _, rr := range resp.Answer {
		a, ok := rr.(*dns.A)
		if !ok {
			continue
		}
		ip := a.A.To4()
		if ip == nil {
			continue
		}
		ips = append(ips, ip)
		d := time.Duration(a.Hdr.Ttl) * time.Second
		if ttl == 0 || d < ttl {
			ttl = d
		}
	}
	if len(ips) == 0 {
		return nil, 0, fmt.Errorf("DoH: no A record for %s", domain)
	}
	return ips, ttl, nil
}
