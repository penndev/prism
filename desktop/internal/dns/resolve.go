package dns

import (
	"net"
	"strings"

	"desktop/internal/storage"

	"github.com/miekg/dns"
)

func resolve(req *dns.Msg) (*dns.Msg, error) {
	if storage.DefaultStorage != nil {
		if cfg, err := storage.DefaultStorage.GetRuleConfig(); err == nil && cfg != nil {
			for _, q := range req.Question {
				if matchDomain(q.Name, cfg.Domains) {
					return queryDoH(req)
				}
			}
		}
	}
	c := &dns.Client{Net: "udp", Timeout: udpTimeout}
	resp, _, err := c.Exchange(req, net.JoinHostPort(currentUpstream(), "53"))
	return resp, err
}

func matchDomain(qname string, domains []string) bool {
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(qname), "."))
	if name == "" {
		return false
	}
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(d), "."))
		if d == "" {
			continue
		}
		if name == d || strings.HasSuffix(name, "."+d) {
			return true
		}
	}
	return false
}
