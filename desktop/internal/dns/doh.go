package dns

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"desktop/internal/storage"
	"github.com/miekg/dns"
)

func queryDoH(req *dns.Msg) (*dns.Msg, error) {
	data, err := req.Pack()
	if err != nil {
		return nil, err
	}
	proxyURL := &url.URL{Scheme: "http", Host: defaultProxy}
	if storage.DefaultStorage != nil {
		if s, err := storage.DefaultStorage.GetSettings(); err == nil && s != nil {
			if s.Proxy.Port > 0 {
				proxyURL.Host = net.JoinHostPort("127.0.0.1", strconv.Itoa(s.Proxy.Port))
			}
			if s.Proxy.Username != "" || s.Proxy.Password != "" {
				proxyURL.User = url.UserPassword(s.Proxy.Username, s.Proxy.Password)
			}
		}
	}
	httpReq, err := http.NewRequest(http.MethodPost, dohUpstream, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")

	httpResp, err := (&http.Client{
		Timeout:   dohTimeout,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}).Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DoH: %s", httpResp.Status)
	}
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	resp := new(dns.Msg)
	return resp, resp.Unpack(body)
}
